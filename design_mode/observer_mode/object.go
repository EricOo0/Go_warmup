package observer_mode

import "fmt"

type Object interface {
	DisplayData()
	Register()
	Update(msg string)
}

// 观察者实例

type ObjectA struct {
	Sub Subject
	Msg string
}

func (obj *ObjectA) DisplayData() {
	fmt.Println("objectA get data,data is:", obj.Msg)
}
func (obj *ObjectA) Register() {
	obj.Sub.AddObject(obj)

}
func (obj *ObjectA) Update(msg string) {
	obj.Msg = msg
	obj.DisplayData()
}
func (obj *ObjectA) OwnFunction(msg string) {
	fmt.Println("This is own function of Object A")
}

type ObjectB struct {
	Sub Subject
	Msg string
}

func (obj *ObjectB) DisplayData() {
	fmt.Println("objectB get data,data is:", obj.Msg)
}
func (obj *ObjectB) Register() {
	obj.Sub.AddObject(obj)

}
func (obj *ObjectB) Update(msg string) {
	obj.Msg = msg
	obj.DisplayData()
}
func (obj *ObjectB) OwnFunction(msg string) {
	fmt.Println("This is own function of Object B")
}
