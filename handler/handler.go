package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ankr-network/dccn-common/protos"
	"github.com/Ankr-network/dccn-common/protos/common"
	"github.com/Ankr-network/dccn-common/protos/taskmgr/v1/micro"
	db "github.com/Ankr-network/dccn-taskmgr/db_service"
	"github.com/google/uuid"
	"github.com/gorhill/cronexpr"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/metadata"
	"gopkg.in/mgo.v2/bson"
	"k8s.io/helm/pkg/chartutil"
)

type TaskMgrHandler struct {
	db         db.DBService
	deployTask micro.Publisher
}

func New(db db.DBService, deployTask micro.Publisher) *TaskMgrHandler {

	return &TaskMgrHandler{
		db:         db,
		deployTask: deployTask,
	}
}

type Token struct {
	Exp int64
	Jti string
	Iss string
}

func getUserID(ctx context.Context) string {
	meta, ok := metadata.FromContext(ctx)
	// Note this is now uppercase (not entirely sure why this is...)
	var token string
	if ok {
		token = meta["token"]
	}

	parts := strings.Split(token, ".")

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		fmt.Println("decode error:", err)

	}
	fmt.Println(string(decoded))

	var dat Token

	if err := json.Unmarshal(decoded, &dat); err != nil {
		panic(err)
	}

	return string(dat.Jti)
}

func (p *TaskMgrHandler) CreateTask(ctx context.Context, req *taskmgr.CreateTaskRequest, rsp *taskmgr.CreateTaskResponse) error {
	uid := getUserID(ctx)
	log.Println("task manager service CreateTask")

	if req.Task.Attributes.Replica < 0 || req.Task.Attributes.Replica >= 100 {
		log.Println(ankr_default.ErrReplicaTooMany)
		return ankr_default.ErrReplicaTooMany
	}

	log.Printf("CreateTask task %+v", req)

	if req.Task.Attributes.Replica == 0 {
		req.Task.Attributes.Replica = 1
	}

	if req.Task.Type == common_proto.TaskType_CRONJOB { // check schudule filed
		_, err := cronexpr.Parse(req.Task.GetTypeCronJob().Schedule)
		if err != nil {
			log.Printf("check crobjob scheducle fomat error %s \n", err.Error())
			return ankr_default.ErrCronJobScheduleFormat
		}

	}

	req.Task.Status = common_proto.TaskStatus_STARTING
	req.Task.Id = uuid.New().String()
	req.Task.Uid = uid
	rsp.TaskId = req.Task.Id

	if req.Task.ChartRepo == "user" {

		path, err := filepath.Abs("chart_repo/" + req.Task.Type.String())
		if err != nil {
			log.Printf("cannot get chart folder path...\n %s \n", err.Error())
			return errors.New("internal error: cannot get chart folder path")
		}

		customChart, err := chartutil.LoadDir(path)
		if err != nil {
			log.Printf("cannot load chart from folder %s ...\n %s \n", path, err.Error())
			return errors.New("internal error: cannot load chart from folder")
		}

		//customChart.Values.Values["replicaCount"] = &chart.Value{Value: fmt.Sprint(req.Task.Attributes.Replica)}
		customChart.Metadata.Version = req.Task.ChartVer
		customChart.Metadata.Name = req.Task.ChartName

		dest, err := os.Getwd()
		if err != nil {
			log.Printf("cannot get chart outdir")
			return errors.New("internal error: cannot get chart outdir")
		}
		log.Printf("save to outdir: %s\n", dest)
		name, err := chartutil.Save(customChart, dest)
		if err == nil {
			log.Printf("Successfully packaged chart and saved it to: %s\n", name)
		} else {
			log.Printf("Failed to save: %s", err)
			return errors.New("internal error: Failed to save chart to outdir")
		}

		file, err := os.Open(name)
		if err != nil {
			log.Printf("cannot open chart zip file")
			return errors.New("internal error: cannot open chart zip file")
		}

		chartReq, err := http.NewRequest("POST", "http://chart-dev.dccn.ankr.network:8080/api/user/"+uid+"/charts", file)
		if err != nil {
			log.Printf("cannot open chart zip file")
			return errors.New("internal error: cannot open chart zip file")
		}

		chartRes, err := http.DefaultClient.Do(chartReq)

		message, _ := ioutil.ReadAll(chartRes.Body)
		defer chartRes.Body.Close()

		log.Printf(string(message))

	}

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_CREATE,
		OpPayload: &common_proto.DCStream_Task{Task: req.Task},
	}

	if err := p.deployTask.Publish(context.Background(), &event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return ankr_default.ErrPublish
	} else {
		log.Println("task manager service send CreateTask MQ message to dc manager service (api)")
	}

	if err := p.db.Create(req.Task, uid); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

// Must return nil for gRPC handler
func (p *TaskMgrHandler) CancelTask(ctx context.Context, req *taskmgr.TaskID, rsp *common_proto.Empty) error {
	userId := getUserID(ctx)
	log.Println("Debug into CancelTask")
	if err := checkId(userId, req.TaskId); err != nil {
		log.Println(err.Error())
		return err
	}
	task, err := p.checkOwner(userId, req.TaskId)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	if task.Status == common_proto.TaskStatus_CANCELLED {
		return ankr_default.ErrCanceledTwice
	}

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_CANCEL,
		OpPayload: &common_proto.DCStream_Task{Task: task},
	}

	if err := p.deployTask.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}

	if err := p.db.Update(task.Id, bson.M{"$set": bson.M{"status": common_proto.TaskStatus_CANCELLED}}); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func convertToTaskMessage(task db.TaskRecord) common_proto.Task {
	message := common_proto.Task{}
	message.Id = task.ID
	message.Name = task.Name
	message.Type = task.Type
	message.Status = task.Status
	message.DataCenterName = task.Datacenter
	message.Attributes = &common_proto.TaskAttributes{}
	message.Attributes.Replica = task.Replica
	message.Attributes.LastModifiedDate = task.Last_modified_date
	message.Attributes.CreationDate = task.Creation_date

	//deployMessage := common_proto.TaskTypeDeployment{Image : task.Image}
	if task.Type == common_proto.TaskType_DEPLOYMENT {
		t := common_proto.Task_TypeDeployment{TypeDeployment: &common_proto.TaskTypeDeployment{Image: task.Image}}
		message.TypeData = &t
	}

	if task.Type == common_proto.TaskType_JOB {
		t := common_proto.Task_TypeJob{TypeJob: &common_proto.TaskTypeJob{Image: task.Image}}
		message.TypeData = &t
	}

	if task.Type == common_proto.TaskType_CRONJOB {
		t := common_proto.Task_TypeCronJob{TypeCronJob: &common_proto.TaskTypeCronJob{Image: task.Image, Schedule: task.Schedule}}
		message.TypeData = &t
	}

	return message

}

func (p *TaskMgrHandler) TaskList(ctx context.Context, req *taskmgr.TaskListRequest, rsp *taskmgr.TaskListResponse) error {
	userId := getUserID(ctx)
	log.Println("task service into TaskList")

	tasks, err := p.db.GetAll(userId)
	log.Printf(">>>>>>taskMessage  %+v \n", tasks)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	tasksWithoutHidden := make([]*common_proto.Task, 0)

	for i := 0; i < len(tasks); i++ {
		if tasks[i].Hidden != true {
			taskMessage := convertToTaskMessage(tasks[i])
			log.Printf("taskMessage  %+v \n", taskMessage)
			tasksWithoutHidden = append(tasksWithoutHidden, &taskMessage)
		}
	}

	rsp.Tasks = tasksWithoutHidden

	return nil
}

func (p *TaskMgrHandler) UpdateTask(ctx context.Context, req *taskmgr.UpdateTaskRequest, rsp *common_proto.Empty) error {
	userId := getUserID(ctx)

	if err := checkId(userId, req.Task.Id); err != nil {
		log.Println(err.Error())
		return err
	}

	task, err := p.checkOwner(userId, req.Task.Id)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	req.Task.Name = strings.ToLower(req.Task.Name)

	if req.Task.Attributes.Replica == 0 {
		req.Task.Attributes.Replica = task.Attributes.Replica
	}

	if req.Task.Attributes.Replica < 0 || req.Task.Attributes.Replica >= 100 {
		log.Println(ankr_default.ErrReplicaTooMany.Error())
		return ankr_default.ErrReplicaTooMany
	}

	if task.Status == common_proto.TaskStatus_CANCELLED ||
		task.Status == common_proto.TaskStatus_DONE {
		log.Println(ankr_default.ErrTaskStatusCanNotUpdate.Error())
		return ankr_default.ErrTaskStatusCanNotUpdate
	}

	task.ChartRepo = req.Task.ChartRepo
	task.ChartName = req.Task.ChartName
	task.ChartVer = req.Task.ChartVer

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_UPDATE,
		OpPayload: &common_proto.DCStream_Task{Task: task},
	}

	if err := p.deployTask.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}
	// TODO: wait deamon notify
	req.Task.Status = common_proto.TaskStatus_UPDATING
	if err := p.db.UpdateTask(req.Task.Id, req.Task); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func (p *TaskMgrHandler) TaskOverview(ctx context.Context, req *common_proto.Empty, rsp *taskmgr.TaskOverviewResponse) error {
	log.Printf("TaskOverview in task manager service\n")
	//rsp = &taskmgr.TaskOverviewResponse{}
	userId := getUserID(ctx)
	tasks, err := p.db.GetAll(userId)
	failed := 0

	for i := 0; i < len(tasks); i++ {
		t := tasks[i]
		if t.Status == common_proto.TaskStatus_START_FAILED || t.Status == common_proto.TaskStatus_CANCEL_FAILED || t.Status == common_proto.TaskStatus_UPDATE_FAILED {
			failed++
		}
	}

	rsp.ClusterCount = 0
	rsp.EnvironmentCount = 0
	rsp.RegionCount = 0
	rsp.TotalTaskCount = 1
	rsp.HealthTaskCount = 1

	if err == nil && len(tasks) > 0 {
		rsp.ClusterCount = int32(len(tasks))
		rsp.EnvironmentCount = int32(len(tasks))
		rsp.RegionCount = 3
		rsp.TotalTaskCount = int32(len(tasks))
		rsp.HealthTaskCount = int32(len(tasks) - failed)

	}

	return nil
}

func (p *TaskMgrHandler) TaskLeaderBoard(ctx context.Context, req *common_proto.Empty, rsp *taskmgr.TaskLeaderBoardResponse) error {
	log.Printf("TaskleaderBoard in task manager service\n")
	//rsp = &taskmgr.TaskLeaderBoardResponse{}
	list := make([]*taskmgr.TaskLeaderBoardDetail, 0)
	{
		detail := taskmgr.TaskLeaderBoardDetail{}
		detail.Name = "task_1"
		detail.Number = 99.34
		list = append(list, &detail)
	}

	{
		detail := taskmgr.TaskLeaderBoardDetail{}
		detail.Name = "task_2"
		detail.Number = 98.53
		list = append(list, &detail)
	}

	{
		detail := taskmgr.TaskLeaderBoardDetail{}
		detail.Name = "task_3"
		detail.Number = 97.98
		list = append(list, &detail)
	}

	userId := getUserID(ctx)
	tasks, err := p.db.GetAll(userId)
	if err == nil && len(tasks) > 0 {
		offset := len(tasks)

		for i := 0; i < len(list); i++ {
			if offset == 0 {
				break
			}
			list[i].Name = tasks[offset-1].Name
			offset--
		}
	}

	rsp.List = list

	log.Printf("TaskleaderBoard  <<<<<>>>>>> %+v", rsp.List)

	return nil

}

func (p *TaskMgrHandler) PurgeTask(ctx context.Context, req *taskmgr.TaskID, rsp *common_proto.Empty) error {
	error := p.CancelTask(ctx, req, rsp)

	if error == ankr_default.ErrCanceledTwice {
		return ankr_default.ErrPurgedTwice
	}

	if error == nil {
		log.Printf(" PurgeTask  %s \n", req.TaskId)
		p.db.Update(req.TaskId, bson.M{"$set": bson.M{"hidden": true}})
	}
	return error
}

func (p *TaskMgrHandler) checkOwner(userId, taskId string) (*common_proto.Task, error) {
	task, err := p.db.Get(taskId)

	if err != nil {
		return nil, err
	}

	log.Printf("taskid : %s user id -%s-   user_token_id -%s-  ", taskId, task.Userid, userId)

	if task.Userid != userId {
		return nil, ankr_default.ErrUserNotOwn
	}

	taskMessage := convertToTaskMessage(task)

	return &taskMessage, nil
}

func checkId(userId, taskId string) error {
	if userId == "" {
		return ankr_default.ErrUserNotExist
	}

	if taskId == "" {
		return ankr_default.ErrUserNotOwn
	}

	return nil
}
