package main

import (
	"strconv"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func Test_isLockFree(t *testing.T) {
	type args struct {
		annotations map[string]string
		now         time.Time
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty cm",
			args: args{
				annotations: map[string]string{},
				now:         time.Now().UTC(),
			},
			wantErr: false,
		},
		{
			name: "no expiry",
			args: args{
				annotations: map[string]string{
					"holder": "foo",
				},
				now: time.Now().UTC(),
			},
			wantErr: false,
		},
		{
			name: "expiry in the past",
			args: args{
				annotations: map[string]string{
					"holder": "foo",
					"expiry": strconv.FormatInt(time.Now().UTC().Add(-10*time.Second).Unix(), 10),
				},
				now: time.Now().UTC(),
			},
			wantErr: false,
		},
		{
			name: "held",
			args: args{
				annotations: map[string]string{
					"holder": "foo",
					"expiry": strconv.FormatInt(time.Now().UTC().Add(10*time.Second).Unix(), 10),
				},
				now: time.Now().UTC(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tt.args.annotations,
				},
			}
			if err := isLockFree(cm, tt.args.now); (err != nil) != tt.wantErr {
				t.Errorf("isLockFree() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
