package core

//这个结构体是对类型的一个封装
type GodisObject struct {
	ObjectType int
	Ptr        interface{} //在这里传递接口来描述类型信息，通用型更强
}

const ObjectTypeString = 1

//创建一个特定类型的object
func CreatObject(objecttype int, ptr interface{}) (o *GodisObject) {
	o = new(GodisObject)
	o.ObjectType = objecttype
	o.Ptr = ptr
	return

}
