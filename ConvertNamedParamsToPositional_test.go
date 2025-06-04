package dbx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertNamedParamsToPositional(t *testing.T) {
	sql := "select *,@a+@b where a=@a and b=@b"
	m := map[string]interface{}{"a": 1, "b": 2}
	sql, args, err := convertNamedParamsToPositional(sql, m)
	assert.NoError(t, err)
	assert.Equal(t, "select *,?+? where a=? and b=?", sql)
	assert.Equal(t, []interface{}{1, 2, 1, 2}, args)
	sql = "select *,sum(x+@b) m  group by y having m>@a+@c where a=@a and b=@b"
	m = map[string]interface{}{"a": 1, "b": 2, "c": 3}
	sql, args, err = convertNamedParamsToPositional(sql, m)
	assert.NoError(t, err)
	assert.Equal(t, "select *,sum(x+?) m  group by y having m>?+? where a=? and b=?", sql)
	assert.Equal(t, []interface{}{2, 1, 3, 1, 2}, args)
	sql = "select *,@salary^2+@bonus m  group by y having m>@a+@c where a=@a and b=@b"
	m = map[string]interface{}{"a": 1, "b": 2, "c": 3, "salary": 4, "bonus": 5}
	sql, args, err = convertNamedParamsToPositional(sql, m)
	assert.NoError(t, err)
	assert.Equal(t, "select *,?^2+? m  group by y having m>?+? where a=? and b=?", sql)
	assert.Equal(t, []interface{}{4, 5, 1, 3, 1, 2}, args)

	t.Log(sql)
}
