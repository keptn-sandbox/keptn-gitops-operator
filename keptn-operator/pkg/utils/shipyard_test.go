package utils

import (
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func Test_composeKeptnStage(t *testing.T) {
	teststage := keptnv1.KeptnStage{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my_first_stage",
		},
		Spec: keptnv1.KeptnStageSpec{
			Project: "my_project",
			Sequence: []keptnv1.KeptnSequenceRefSpec{
				{
					Type:        "sequenceref",
					SequenceRef: "my_first_sequence",
				},
			},
		},
	}

	testsequencelist_correct := &keptnv1.KeptnSequenceList{
		Items: []keptnv1.KeptnSequence{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my_first_sequence",
				},
				Spec: keptnv1.KeptnSequenceSpec{
					Sequence: keptnv1.Sequence{
						Name: "my_first_sequence",
						Tasks: []keptnv1.Task{
							{
								Name: "artifact-delivery",
							},
							{
								Name: "promotion",
							},
						},
					},
				},
			},
		},
	}

	testsequencelist_missing := &keptnv1.KeptnSequenceList{
		Items: []keptnv1.KeptnSequence{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my_first_sequencex",
				},
				Spec: keptnv1.KeptnSequenceSpec{
					Sequence: keptnv1.Sequence{
						Name: "my_first_sequence",
						Tasks: []keptnv1.Task{
							{
								Name: "artifact-delivery",
							},
							{
								Name: "promotion",
							},
						},
					},
				},
			},
		},
	}

	testresult := keptnv1.Stage{
		Name: "my_first_stage",
		Sequences: []keptnv1.Sequence{
			{
				Name: "my_first_sequence",
				Tasks: []keptnv1.Task{
					{
						Name: "artifact-delivery",
					},
					{
						Name: "promotion",
					},
				},
			},
		},
	}
	type args struct {
		stage     keptnv1.KeptnStage
		sequences *keptnv1.KeptnSequenceList
	}
	tests := []struct {
		name    string
		args    args
		want    keptnv1.Stage
		wantErr bool
	}{
		{
			name: "stage_correct",
			args: args{
				stage:     teststage,
				sequences: testsequencelist_correct,
			},
			want: testresult,
		},
		{
			name: "stage_missing_sequence",
			args: args{
				stage:     teststage,
				sequences: testsequencelist_missing,
			},
			want: keptnv1.Stage{
				Name: teststage.Name,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := composeKeptnStage(tt.args.stage, tt.args.sequences)
			if (err != nil) != tt.wantErr {
				t.Errorf("composeKeptnStage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("composeKeptnStage() got = %v, want %v", got, tt.want)
			}
		})
	}
}
