package decorator_mode

import "fmt"

type Base interface {
	DoSomething()
}
type Object struct{}

func (obj *Object) DoSomething() {
	fmt.Println("obj do something")
}

type ObjDecorator struct {
	obj Object
}

func (deco *ObjDecorator) DoSomething() {
	fmt.Println("Decorator Do something")
	deco.obj.DoSomething()
}
