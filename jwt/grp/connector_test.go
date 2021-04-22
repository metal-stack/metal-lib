package grp

import (
	"testing"
)

func TestParseConnectorId(t *testing.T) {
	type args struct {
		connectorId string
	}
	tests := []struct {
		name          string
		args          args
		wantJwtTenant string
		wantDirectory string
		wantErr       bool
	}{
		{
			name: "empty",
			args: args{
				connectorId: "",
			},
			wantJwtTenant: "",
			wantDirectory: "",
			wantErr:       true,
		},
		{
			name: "no separator",
			args: args{
				connectorId: "tnnt",
			},
			wantJwtTenant: "",
			wantDirectory: "",
			wantErr:       true,
		},
		{
			name: "only separator",
			args: args{
				connectorId: "_",
			},
			wantJwtTenant: "",
			wantDirectory: "",
			wantErr:       false,
		},
		{
			name: "tnnt_ldap",
			args: args{
				connectorId: "tnnt_ldap",
			},
			wantJwtTenant: "tnnt",
			wantDirectory: "ldap",
			wantErr:       false,
		},
		{
			name: "ddd_ad",
			args: args{
				connectorId: "ddd_ad",
			},
			wantJwtTenant: "ddd",
			wantDirectory: "ad",
			wantErr:       false,
		},
		{
			name: "ddd_ldap_openldap",
			args: args{
				connectorId: "ddd_ldap_openldap",
			},
			wantJwtTenant: "ddd",
			wantDirectory: "ldap",
			wantErr:       false,
		},
		{
			name: "ddd_ldap_openldap_something",
			args: args{
				connectorId: "ddd_ldap_openldap_something",
			},
			wantJwtTenant: "ddd",
			wantDirectory: "ldap",
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotJwtTenant, gotDirectory, err := ParseConnectorId(tt.args.connectorId)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConnectorId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotJwtTenant != tt.wantJwtTenant {
				t.Errorf("ParseConnectorId() gotJwtTenant = %v, want %v", gotJwtTenant, tt.wantJwtTenant)
			}
			if gotDirectory != tt.wantDirectory {
				t.Errorf("ParseConnectorId() gotDirectory = %v, want %v", gotDirectory, tt.wantDirectory)
			}
		})
	}
}
