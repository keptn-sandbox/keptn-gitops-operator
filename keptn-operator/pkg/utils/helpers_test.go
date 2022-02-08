package utils

import (
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"io"
	nethttp "net/http"
	"strings"
	"testing"
)

func TestContainsString(t *testing.T) {
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "contains_string",
			args: args{
				slice: []string{
					"Test1",
					"Test2",
				},
				s: "Test1",
			},
			want: true,
		},
		{
			name: "not_contains_string",
			args: args{
				slice: []string{
					"Test1",
					"Test2",
				},
				s: "Test3",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.args.slice, tt.args.s); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckResponseCode(t *testing.T) {
	type args struct {
		response     *nethttp.Response
		expectedCode int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "correct_response",
			args: args{
				response: &nethttp.Response{
					StatusCode: 200,
				},
				expectedCode: 200,
			},
			wantErr: false,
		},
		{
			name: "incorrect_response",
			args: args{
				response: &nethttp.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("body")),
				},
				expectedCode: 300,
			},
			wantErr: true,
		},
		{
			name: "incorrect_response_empty_body",
			args: args{
				response: &nethttp.Response{
					StatusCode: 200,
				},
				expectedCode: 300,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckResponseCode(tt.args.response, tt.args.expectedCode); (err != nil) != tt.wantErr {
				t.Errorf("CheckResponseCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetHashStructure(t *testing.T) {
	testdata := keptnv1.KeptnProject{
		Spec: keptnv1.KeptnProjectSpec{
			Repository:    "my_repository",
			Username:      "keptntester",
			Password:      "keptnpassword",
			DefaultBranch: "my_branch",
		},
	}

	testhash := "17721150619434961566"

	type args struct {
		i interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "correct_struct",
			args: args{i: testdata.Spec},
			want: testhash,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHashStructure(tt.args.i); got != tt.want {
				t.Errorf("GetHashStructure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareHashStructure(t *testing.T) {
	testdata := keptnv1.KeptnProject{
		Spec: keptnv1.KeptnProjectSpec{
			Repository:    "my_repository",
			Username:      "keptntester",
			Password:      "keptnpassword",
			DefaultBranch: "my_branch",
		},
	}
	testdataFail := keptnv1.KeptnProject{
		Spec: keptnv1.KeptnProjectSpec{
			Repository:    "my_repositoryx",
			Username:      "keptntesterx",
			Password:      "keptnpasswordx",
			DefaultBranch: "my_branchx",
		},
	}

	t.Run("compare_structs_correct", func(t *testing.T) {
		if GetHashStructure(testdata.Spec) != GetHashStructure(testdata.Spec) {
			t.Errorf("GetHashStructure() = %v != %v", GetHashStructure(testdata), GetHashStructure(testdata))
		}
	})

	t.Run("compare_structs_false", func(t *testing.T) {
		if GetHashStructure(testdata) == GetHashStructure(testdataFail) {
			t.Errorf("GetHashStructure() = %v == %v", GetHashStructure(testdata), GetHashStructure(testdataFail))
		}
	})
}
