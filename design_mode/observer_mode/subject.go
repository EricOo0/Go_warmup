package observer_mode

// Push mode

type Subject interface {
	AddObject(obj Object)
	DeleteObject(obj Object)
	NotifyObjects()
}

//观察者实例

type RealSubject struct {
	ObjArr []Object
}

func (sub *RealSubject) AddObject(obj Object) {
	sub.ObjArr = append(sub.ObjArr, obj)
}

func (sub *RealSubject) DeleteObject(obj Object) {
	for i, v := range sub.ObjArr {
		if v == obj {
			sub.ObjArr = append(sub.ObjArr[:i], sub.ObjArr[i+1:]...)
		}
	}
}
func (sub *RealSubject) NotifyObjects() {
	for _, obj := range sub.ObjArr {
		msg := "inform msg from subject"
		obj.Update(msg)
	}
}
