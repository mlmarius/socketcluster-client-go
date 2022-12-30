package tests

import (
	"github.com/mlmarius/socketcluster-client-go/scclient/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShouldReturnIncrementedValue(t *testing.T) {
	counter := utils.AtomicCounter{
		Counter: 0,
	}

	actualValue := counter.IncrementAndGet()

	assert.Equal(t, uint64(1), actualValue)
	assert.Equal(t, uint64(1), counter.Value())
}

func TestShouldGetAndIncrementValue(t *testing.T) {
	counter := utils.AtomicCounter{
		Counter: 0,
	}

	actualValue := counter.GetAndIncrement()

	assert.Equal(t, uint64(0), actualValue)
	assert.Equal(t, uint64(1), counter.Value())
}
