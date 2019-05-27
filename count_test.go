package routine

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCount(t *testing.T) {
	bf := bytes.NewBuffer([]byte{})
	assert.NotNil(t, bf)
	rdbf := NewCountReader(bf)
	assert.NotNil(t, rdbf)
	wrbf := NewCountWriter(bf)
	assert.NotNil(t, wrbf)
	n, err := wrbf.Write([]byte("abc"))
	assert.Nil(t, err)
	assert.Equal(t, 3, n)

	p := make([]byte, 1024)
	n, err = rdbf.Read(p)
	assert.Nil(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, rdbf.Count(), wrbf.Count())
}
