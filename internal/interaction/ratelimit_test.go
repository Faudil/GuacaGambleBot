package interaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanAnimateEditBudget(t *testing.T) {
	ResetAnimationBudget()
	SetAnimationBudget(3)
	defer ResetAnimationBudget()

	assert.True(t, CanAnimateEdit())
	assert.True(t, CanAnimateEdit())
	assert.True(t, CanAnimateEdit())
	assert.False(t, CanAnimateEdit(), "budget of 3 edits per second must be enforced")
}

func TestCanAnimateEditClampsBudget(t *testing.T) {
	SetAnimationBudget(0)
	assert.True(t, CanAnimateEdit())
	ResetAnimationBudget()
}
