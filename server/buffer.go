// Copyright 2017 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package server

import (
	"github.com/open-policy-agent/opa/server/controller"
)

// Buffer defines an interface that the server can call to push diagnostic
// information about policy decisions. Buffers must be able to handle
// concurrent calls.
type Buffer = controller.Buffer

// Info contains information describing a policy decision.
type Info = controller.Info

