// Copyright © 2019 Hedzr Yeh.

package exterr

import "bytes"

func New(msg string, errors ...error) error {
	return add(msg, errors...)
}

func add(msg string, errs ...error) error {
	if len(errs) == 0 {
		return &ExtErr{msg: msg}
	} else if len(errs) == 1 {
		err := errs[0]
		if e, ok := err.(*ExtErr); ok {
			return &ExtErr{msg: msg, innerEE: e}
		}
		return &ExtErr{msg: msg, innerErr: err}
	}

	return add(msg, errs[1:]...)
}

func NewError(errors ...error) error {
	return addE(errors...)
}

func addE(errs ...error) error {
	if len(errs) == 0 {
		return &ExtErr{msg: "unknown error"}
	} else if len(errs) == 1 {
		err := errs[0]
		if e, ok := err.(*ExtErr); ok {
			return &ExtErr{innerEE: e}
		}
		return &ExtErr{innerErr: err}
	}

	return addE(errs[1:]...)
}

// func NewEE(err error) error {
// 	if e, ok := err.(*ExtErr); ok {
// 		return &ExtErr{innerEE: e}
// 	}
// 	return &ExtErr{innerErr: err}
// }
// 
// func NewEEM(msg string, err error) error {
// 	if e, ok := err.(*ExtErr); ok {
// 		return &ExtErr{msg: msg, innerEE: e}
// 	}
// 	return &ExtErr{msg: msg, innerErr: err}
// }
// 
// func NewEEE(err, outerErr error) error {
// 	e := &ExtErr{innerErr: outerErr, innerEE: &ExtErr{innerErr: err}}
// 	return e
// }
// 
// func NewEEEM(msg string, err, outerErr error) error {
// 	e := &ExtErr{msg: msg, innerErr: outerErr, innerEE: &ExtErr{innerErr: err}}
// 	return e
// }

type ExtErr struct {
	innerEE  *ExtErr
	innerErr error
	msg      string
}

func (e *ExtErr) Error() string {
	var buf bytes.Buffer
	if len(e.msg) == 0 {
		buf.WriteString("error")
	} else {
		buf.WriteString(e.msg)
	}
	if e.innerErr != nil {
		// buf.WriteString("[")
		buf.WriteString(", ")
		buf.WriteString(e.innerErr.Error())
		// buf.WriteString("]")
	}
	if e.innerEE != nil {
		buf.WriteString("[")
		buf.WriteString(e.innerEE.Error())
		buf.WriteString("]")
	}
	return buf.String()
}
