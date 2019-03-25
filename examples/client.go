package main

import (
	"log"

	grpc "github.com/micro/go-grpc"
)

var token = "token"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("client service start...")
	srv := grpc.NewService()

	srv.Init()
	/*
		cl := taskmgr.NewTaskMgrService(ankr_default.TaskMgrRegistryServerName, srv.Client())

		tokenContext := metadata.NewContext(context.Background(), map[string]string{
			"Token": token,
		})

		tasks := testCommon.MockTasks()
		for i := range tasks {
			if _, err := cl.CreateTask(tokenContext, &taskmgr.CreateTaskRequest{UserId: tasks[i].UserId, Task: &tasks[i]}); err != nil {
				log.Fatal(err.Error())
			} else {
				log.Println("CreateTask Ok")
			}
		}

		userTasks := []*common_proto.Task{}
		if rsp, err := cl.TaskList(tokenContext, &taskmgr.ID{UserId: "1"}); err != nil {
			log.Fatal(err.Error())
		} else {
			userTasks = append(userTasks, rsp.Tasks...)
			log.Println("TaskList Ok")
		}

		if len(userTasks) == 0 {
			log.Fatalf("no tasks belongs to %d\n", 1)
		}

		// CancelTask
		cancelTask := userTasks[0]
		if _, err := cl.CancelTask(tokenContext, &taskmgr.Request{UserId: cancelTask.UserId, TaskId: cancelTask.Id}); err != nil {
			log.Fatal(err.Error())
		} else {
			log.Println("CancelTask Ok")
		}

		// Verify Canceled task
		if _, err := cl.TaskDetail(tokenContext, &taskmgr.Request{UserId: cancelTask.UserId, TaskId: cancelTask.Id}); err != nil {
			log.Fatal(err.Error())
		} else {
			log.Println("TaskDetail Ok")
		}

		// UpdateTask
		cancelTask.Name = "updateTask"
		if _, err := cl.UpdateTask(tokenContext, &taskmgr.UpdateTaskRequest{UserId: cancelTask.UserId, Task: cancelTask}); err != nil {
			log.Fatal(err.Error())
		} else {
			log.Println("TaskDetail Ok")
		}

		/* Verify updated task
		if rsp, err := cl.TaskDetail(tokenContext, &taskmgr.Request{UserId: cancelTask.UserId, TaskId: cancelTask.Id}); err != nil {
			log.Fatal(err.Error())
		} else {
			if !testCommon.IsEqual(rsp.Task, cancelTask) || rsp.Task.Status != common_proto.TaskStatus_UPDATING {
				log.Fatal("UpdateTask operation does not take effect")
			}
			log.Println("UpdateTask takes effect")
		}*/
}
