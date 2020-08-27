package oracle

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/tokenized/specification/dist/golang/actions"
)

func TestEntityIsSubset(t *testing.T) {
	tests := []struct {
		name      string
		sub, full *actions.EntityField
		err       error
	}{
		{
			name: "Simple",
			sub: &actions.EntityField{
				Name: "Tokenized",
			},
			full: &actions.EntityField{
				Name:        "Tokenized",
				CountryCode: "AUS",
			},
			err: nil,
		},
		{
			name: "Simple Fail",
			sub: &actions.EntityField{
				Name: "Tokenized LLC",
			},
			full: &actions.EntityField{
				Name:        "Tokenized",
				CountryCode: "AUS",
			},
			err: errors.New("Name doesn't match"),
		},
		{
			name: "Extra field",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
			},
			full: &actions.EntityField{
				Name:        "Tokenized",
				CountryCode: "AUS",
			},
			err: errors.New("EmailAddress doesn't match"),
		},
		{
			name: "Email",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
			},
			err: nil,
		},
		{
			name: "Administration",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			err: nil,
		},
		{
			name: "Administration type",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 3,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			err: errors.New("Administrator 0"),
		},
		{
			name: "Extra Administration",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
			},
			err: errors.New("Administrator 0"),
		},
		{
			name: "Second Administration",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 5,
						Name: "John",
					},
					&actions.AdministratorField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			err: nil,
		},
		{
			name: "Out of order Administration",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
					&actions.AdministratorField{
						Type: 5,
						Name: "John",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
				Administration: []*actions.AdministratorField{
					&actions.AdministratorField{
						Type: 5,
						Name: "John",
					},
					&actions.AdministratorField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			err: nil,
		},
		{
			name: "Management",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			err: nil,
		},
		{
			name: "Management type",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 1,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			err: errors.New("Manager 0"),
		},
		{
			name: "Extra Management",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
			},
			err: errors.New("Manager 0"),
		},
		{
			name: "Second Management",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 5,
						Name: "John",
					},
					&actions.ManagerField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			err: nil,
		},
		{
			name: "Out of order Management",
			sub: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
					&actions.ManagerField{
						Type: 5,
						Name: "John",
					},
				},
			},
			full: &actions.EntityField{
				Name:         "Tokenized",
				EmailAddress: "satoshi@tokenized.com",
				CountryCode:  "AUS",
				Management: []*actions.ManagerField{
					&actions.ManagerField{
						Type: 5,
						Name: "John",
					},
					&actions.ManagerField{
						Type: 2,
						Name: "Satoshi Nakamoto",
					},
				},
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.sub.Validate(); err != nil {
				t.Errorf("Sub invalid : %s", err)
				return
			}
			if err := tt.full.Validate(); err != nil {
				t.Errorf("Full invalid : %s", err)
				return
			}

			err := VerifyEntityIsSubset(tt.sub, tt.full)

			if tt.err == nil {
				if err != nil {
					t.Errorf("Entity didn't verify : %s", err)
				}
			} else {
				if err == nil {
					t.Errorf("Entity should not have verified : expected %s", tt.err)
				} else if err.Error() != tt.err.Error() {
					t.Errorf("Wrong error : got %s, want %s", err, tt.err)
				}
			}
		})
	}
}
