package sql

import (
	"context"
	"testing"

	"github.com/Jeffail/benthos/v3/public/service"
	"github.com/stretchr/testify/require"
)

func TestSQLSelectInputEmptyShutdown(t *testing.T) {
	conf := `
driver: meow
dsn: woof
table: quack
columns: [ foo, bar, baz ]
where: foo = ?
args_mapping: 'root = [ this.id ]'
`

	spec := sqlSelectInputConfig()
	env := service.NewEnvironment()

	selectConfig, err := spec.ParseYAML(conf, env)
	require.NoError(t, err)

	selectInput, err := newSQLSelectInputFromConfig(selectConfig, nil)
	require.NoError(t, err)
	require.NoError(t, selectInput.Close(context.Background()))
}
