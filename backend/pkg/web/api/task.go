package api

import (
	"context"
	"database/sql"
	"strings"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
	"github.com/valentin-kaiser/hdns/pkg/tasks"
)

// taskToProto converts a stored task into its proto representation. Header
// values are never returned populated to avoid exposing secrets.
func taskToProto(t *schema.Task) *service.Task {
	if t == nil {
		return nil
	}
	proto := &service.Task{
		Id:                 t.ID,
		CreatedAt:          t.CreatedAt.Time.UnixMilli(),
		UpdatedAt:          t.UpdatedAt.Time.UnixMilli(),
		RecordId:           t.RecordID,
		Name:               t.Name,
		TriggerOn:          service.TaskTrigger(t.TriggerOn),
		Method:             t.Method,
		Url:                t.Url,
		Body:               t.Body.String,
		Enabled:            t.Enabled,
		IncludeCertificate: t.IncludeCertificate,
		CertificateFormat:  t.CertificateFormat,
		LastStatus:         t.LastStatus.String,
		LastError:          t.LastError.String,
	}
	if t.LastRun.Valid {
		proto.LastRun = t.LastRun.Time.UnixMilli()
	}
	return proto
}

func (s *Server) GetTasks(ctx context.Context, _ *service.Empty) (*service.TaskList, error) {
	var stored []*schema.Task
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		stored, qerr = q.ListTasks(ctx)
		return qerr
	})
	if err != nil {
		return nil, apperror.NewError("failed to fetch tasks from database").AddError(err)
	}

	list := &service.TaskList{}
	for _, t := range stored {
		list.Tasks = append(list.Tasks, taskToProto(t))
	}
	return list, nil
}

func (s *Server) UpsertTask(ctx context.Context, in *service.Task) (*service.Task, error) {
	if in == nil {
		return nil, apperror.NewError("task is required")
	}
	if in.RecordId == 0 {
		return nil, apperror.NewError("task record id is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperror.NewError("task name is required")
	}
	if strings.TrimSpace(in.Url) == "" {
		return nil, apperror.NewError("task url is required")
	}

	method := strings.ToUpper(strings.TrimSpace(in.Method))
	if method == "" {
		method = "POST"
	}

	body := sql.NullString{String: in.Body, Valid: in.Body != ""}
	certFormat := tasks.ResolveCertificateFormat(in.CertificateFormat, in.Body)

	var task *schema.Task
	err := database.HDNS().Query(func(q *schema.Queries) error {
		switch in.Id {
		case 0:
			headers, err := tasks.EncryptHeaders(in.Headers)
			if err != nil {
				return apperror.NewError("invalid task headers").AddError(err)
			}

			id, cerr := q.CreateTask(ctx, schema.CreateTaskParams{
				RecordID:           in.RecordId,
				Name:               in.Name,
				TriggerOn:          int8(in.TriggerOn),
				Method:             method,
				Url:                in.Url,
				Headers:            headers,
				Body:               body,
				Enabled:            in.Enabled,
				IncludeCertificate: in.IncludeCertificate,
				CertificateFormat:  certFormat,
			})
			if cerr != nil {
				return apperror.NewError("failed to create task in database").AddError(cerr)
			}
			in.Id = id
		default:
			task, err := q.GetTask(ctx, in.Id)
			if err != nil {
				return apperror.NewError("failed to fetch existing task from database").AddError(err)
			}

			headers := task.Headers
			if in.Headers != "" {
				var err error
				headers, err = tasks.EncryptHeaders(in.Headers)
				if err != nil {
					return apperror.NewError("invalid task headers").AddError(err)
				}
			}

			uerr := q.UpdateTask(ctx, schema.UpdateTaskParams{
				ID:                 in.Id,
				RecordID:           in.RecordId,
				Name:               in.Name,
				TriggerOn:          int8(in.TriggerOn),
				Method:             method,
				Url:                in.Url,
				Headers:            headers,
				Body:               body,
				Enabled:            in.Enabled,
				IncludeCertificate: in.IncludeCertificate,
				CertificateFormat:  certFormat,
			})
			if uerr != nil {
				return apperror.NewError("failed to update task in database").AddError(uerr)
			}
		}

		var gerr error
		task, gerr = q.GetTask(ctx, in.Id)
		return gerr
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	return taskToProto(task), nil
}

func (s *Server) DeleteTask(ctx context.Context, in *service.TaskDelete) (*service.Empty, error) {
	if in == nil || in.Id == 0 {
		return nil, apperror.NewError("task id is required")
	}

	err := database.HDNS().Query(func(q *schema.Queries) error {
		return q.DeleteTask(ctx, in.Id)
	})
	if err != nil {
		return nil, apperror.NewError("failed to delete task from database").AddError(err)
	}

	return &service.Empty{}, nil
}

func (s *Server) RunTask(ctx context.Context, in *service.Task) (*service.TaskResult, error) {
	if in == nil {
		return nil, apperror.NewError("task is required")
	}

	if in.Id == 0 {
		return nil, apperror.NewError("task id is required")
	}

	var task *schema.Task
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		task, err = q.GetTask(ctx, in.Id)
		if err != nil {
			return apperror.NewError("failed to fetch task from database").AddError(err)
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	status, err := tasks.RunTask(ctx, &schema.Task{
		ID:                 task.ID,
		RecordID:           task.RecordID,
		Name:               task.Name,
		Method:             task.Method,
		Url:                task.Url,
		Headers:            task.Headers,
		Body:               sql.NullString{String: task.Body.String, Valid: task.Body.Valid},
		IncludeCertificate: task.IncludeCertificate,
		CertificateFormat:  task.CertificateFormat,
	})

	result := &service.TaskResult{Status: status}
	if err != nil {
		result.Error = err.Error()
	}
	return result, nil
}
