package swarmcd

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
)

func TestDeriveHealth(t *testing.T) {
	tests := []struct {
		name    string
		running uint64
		desired uint64
		mode    string
		want    string
	}{
		{"replicated all running", 3, 3, "replicated", HealthHealthy},
		{"replicated partial", 1, 3, "replicated", HealthProgressing},
		{"replicated none running", 0, 3, "replicated", HealthDegraded},
		{"replicated zero desired", 0, 0, "replicated", HealthProgressing},
		{"global all running", 2, 2, "global", HealthHealthy},
		{"global one down", 1, 2, "global", HealthProgressing},
		{"global none running", 0, 2, "global", HealthDegraded},
		{"running exceeds desired", 4, 3, "replicated", HealthHealthy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveHealth(tt.running, tt.desired, tt.mode); got != tt.want {
				t.Errorf("deriveHealth(%d, %d, %q) = %q, want %q", tt.running, tt.desired, tt.mode, got, tt.want)
			}
		})
	}
}

func TestStripImageDigest(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"nginx:1.27", "nginx:1.27"},
		{"nginx:1.27@sha256:abcdef", "nginx:1.27"},
		{"registry.example.com/app@sha256:deadbeef", "registry.example.com/app"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := stripImageDigest(tt.in); got != tt.want {
			t.Errorf("stripImageDigest(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestServiceToStatus(t *testing.T) {
	replicas := uint64(3)
	svc := swarm.Service{
		ID: "abc123",
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{Name: "web_nginx"},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{Image: "nginx:1.27@sha256:deadbeef"},
			},
			Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
		},
		ServiceStatus: &swarm.ServiceStatus{RunningTasks: 2, DesiredTasks: 3},
	}

	got := serviceToStatus(svc)
	if got.ID != "abc123" || got.Name != "web_nginx" {
		t.Fatalf("unexpected identity mapping: %+v", got)
	}
	if got.Image != "nginx:1.27" {
		t.Errorf("Image = %q, want %q (digest should be stripped)", got.Image, "nginx:1.27")
	}
	if got.Mode != "replicated" {
		t.Errorf("Mode = %q, want %q", got.Mode, "replicated")
	}
	if got.RunningTasks != 2 || got.DesiredTasks != 3 {
		t.Errorf("replica counts = %d/%d, want 2/3", got.RunningTasks, got.DesiredTasks)
	}
	if got.Health != HealthProgressing {
		t.Errorf("Health = %q, want %q", got.Health, HealthProgressing)
	}
}

func TestServiceToStatus_GlobalNoServiceStatus(t *testing.T) {
	svc := swarm.Service{
		ID: "g1",
		Spec: swarm.ServiceSpec{
			Annotations:  swarm.Annotations{Name: "mon_agent"},
			TaskTemplate: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: "agent:latest"}},
			Mode:         swarm.ServiceMode{Global: &swarm.GlobalService{}},
		},
	}

	got := serviceToStatus(svc)
	if got.Mode != "global" {
		t.Errorf("Mode = %q, want %q", got.Mode, "global")
	}
	// No ServiceStatus and no replicated replicas -> desired 0 -> progressing.
	if got.Health != HealthProgressing {
		t.Errorf("Health = %q, want %q", got.Health, HealthProgressing)
	}
	// Secrets/Configs must serialise as arrays, never nil.
	if got.Secrets == nil || got.Configs == nil || got.Tasks == nil {
		t.Errorf("Secrets/Configs/Tasks must be non-nil: %+v", got)
	}
}

func TestTaskToStatus(t *testing.T) {
	task := swarm.Task{
		ID:     "task-1",
		Slot:   2,
		NodeID: "node-abc",
		Status: swarm.TaskStatus{
			State:           swarm.TaskStateRunning,
			ContainerStatus: &swarm.ContainerStatus{ContainerID: "c0ffee"},
		},
		DesiredState: swarm.TaskStateRunning,
	}

	got := taskToStatus(task)
	if got.ID != "task-1" || got.Slot != 2 || got.NodeID != "node-abc" {
		t.Fatalf("unexpected identity mapping: %+v", got)
	}
	if got.State != "running" || got.DesiredState != "running" {
		t.Errorf("state mapping = %q/%q, want running/running", got.State, got.DesiredState)
	}
	if got.ContainerID != "c0ffee" {
		t.Errorf("ContainerID = %q, want %q", got.ContainerID, "c0ffee")
	}
}

func TestTaskToStatus_NoContainerStatus(t *testing.T) {
	task := swarm.Task{
		ID:           "task-2",
		Status:       swarm.TaskStatus{State: swarm.TaskStateFailed, Err: "boom"},
		DesiredState: swarm.TaskStateShutdown,
	}

	got := taskToStatus(task)
	if got.ContainerID != "" {
		t.Errorf("ContainerID = %q, want empty", got.ContainerID)
	}
	if got.State != "failed" || got.Error != "boom" {
		t.Errorf("got state=%q err=%q, want failed/boom", got.State, got.Error)
	}
}

func TestServiceSecretRefs(t *testing.T) {
	svc := swarm.Service{
		Spec: swarm.ServiceSpec{
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Secrets: []*swarm.SecretReference{{SecretName: "tls-cert"}, {SecretName: "tls-key"}},
					Configs: []*swarm.ConfigReference{{ConfigName: "app-config"}},
				},
			},
		},
	}

	secrets, configs := serviceSecretRefs(svc)
	if len(secrets) != 2 || secrets[0] != "tls-cert" || secrets[1] != "tls-key" {
		t.Errorf("secrets = %v, want [tls-cert tls-key]", secrets)
	}
	if len(configs) != 1 || configs[0] != "app-config" {
		t.Errorf("configs = %v, want [app-config]", configs)
	}
}

func TestServiceSecretRefs_None(t *testing.T) {
	svc := swarm.Service{Spec: swarm.ServiceSpec{TaskTemplate: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{}}}}

	secrets, configs := serviceSecretRefs(svc)
	if secrets == nil || configs == nil {
		t.Fatalf("expected non-nil slices, got secrets=%v configs=%v", secrets, configs)
	}
	if len(secrets) != 0 || len(configs) != 0 {
		t.Errorf("expected empty slices, got secrets=%v configs=%v", secrets, configs)
	}
}

func TestIsFailedTaskState(t *testing.T) {
	for _, state := range []swarm.TaskState{swarm.TaskStateFailed, swarm.TaskStateRejected, swarm.TaskStateOrphaned} {
		if !isFailedTaskState(state) {
			t.Errorf("isFailedTaskState(%q) = false, want true", state)
		}
	}
	for _, state := range []swarm.TaskState{swarm.TaskStateRunning, swarm.TaskStateShutdown, swarm.TaskStateComplete, swarm.TaskStateStarting} {
		if isFailedTaskState(state) {
			t.Errorf("isFailedTaskState(%q) = true, want false", state)
		}
	}
}
