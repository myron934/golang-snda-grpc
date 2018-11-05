package inject

import (
	"fmt"
	"reflect"
	"testing"
)

type IFoo interface {
	Do()
}
type Foo struct {
	X string
	Y int
}

func (f *Foo) Do() {
	fmt.Printf("X is: %s, Y is: %d\n", f.X, f.Y)

}

func (f *Foo) Sum(n int) int {
	f.Y += n
	fmt.Printf("X is: %s, Y is: %d\n", f.X, f.Y)
	return f.Y
}

func Test1(t *testing.T) {
	var s string = "abc"
	fmt.Println(reflect.TypeOf(&s).String()) //string
	fmt.Println(reflect.TypeOf(s).Name())    //string

	var f Foo
	typ := reflect.TypeOf(f)
	fmt.Println(typ.String()) //main.Foo
	fmt.Println(typ.Name())   //Foo ，返回结构体的名字
}

func TestImplements(t *testing.T) {
	itp := reflect.TypeOf((*IFoo)(nil)).Elem()
	val := reflect.New(itp).Interface()
	ht := reflect.TypeOf(val).Elem()

	fmt.Println(ht.Kind())
	var f *Foo

	typ := reflect.TypeOf(f)
	fmt.Println(typ.Implements(ht))
}

//获取成员变量
func Test2(t *testing.T) {
	var f Foo
	typ := reflect.TypeOf(f)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fmt.Printf("%s type is :%s\n", field.Name, field.Type)
	}
	field2, ok := typ.FieldByName("X") //等价于typ.Field(0)，返回的也是StructField对象
	if ok {
		fmt.Println(field2.Name)
	}

}

//获取方法, 获取不到私有方法
func Test3(t *testing.T) {
	var f Foo
	typ := reflect.TypeOf(&f)

	fmt.Println(typ.NumMethod()) //1， Foo 方法的个数
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		fmt.Println("====")
		fmt.Println(m.Name) //do
		fmt.Println(m.Type) //func(main.Foo)
		fmt.Println(m.Func) //<func(main.Foo) Value>, 这个返回的是reflect.Value对象，后面再讲
	}

}

//Kind
func Test4(t *testing.T) {
	var f = Foo{}
	typ := reflect.TypeOf(f)
	fmt.Println(typ)        //main.Foo
	fmt.Println(typ.Kind()) //struct

	var f2 = &Foo{}
	typ2 := reflect.TypeOf(f2)
	fmt.Println(typ2)        //*main.Foo
	fmt.Println(typ2.Kind()) //ptr

}

//Value
func Test5(t *testing.T) {
	var i int = 123
	var f = Foo{"abc", 123}
	var s = "abc"
	fmt.Println(reflect.ValueOf(i)) //<int Value>
	fmt.Println(reflect.ValueOf(f)) //<main.Foo Value>
	fmt.Println(reflect.ValueOf(s)) //abc

	//Value.String()方法对string类型的数据做了特殊处理，会直接返回字符串的值。
	//其它类型对象返回的格式都是"<Type% Value>"

}

//Value.Interface
func Test6(t *testing.T) {
	var i int = 123
	fmt.Println(reflect.ValueOf(i).Interface()) //123

	var f = Foo{"abc", 123}
	fmt.Println(f)                                   //{abc 123}
	fmt.Println(reflect.ValueOf(f).Interface() == f) //true
	fmt.Println(reflect.ValueOf(f).Interface())      //{abc 123}

}

//Value.Field
func Test7(t *testing.T) {
	var f = Foo{"abc", 123}
	rv := reflect.ValueOf(f)
	rt := reflect.TypeOf(f)
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		fmt.Printf("%s type is :%s ,value is %v\n", ft.Name, fv.Type(), fv.Interface())
	}
	//X type is :string ,value is abc
	//Y type is :int ,value is 123
}

//赋值
func Test8(t *testing.T) {
	var i int = 123
	fv := reflect.ValueOf(i)
	fe := reflect.ValueOf(&i).Elem() //必须是指针的Value才能调用Elem, 这里拿到了i的一个引用
	fmt.Println(fe)                  //<int Value>
	fmt.Println(fv)                  //<int Value>
	fmt.Println(fv == fe)            //false

	fmt.Println(fe.CanSet()) //true
	fe.SetInt(456)
	fmt.Println(i) //456

}

////赋值
func Test9(t *testing.T) {
	var i = 123
	rv := reflect.ValueOf(&i).Elem()   //这里拿到了i的Value
	fmt.Println(rv.CanSet())           //true
	pi := rv.Addr().Interface().(*int) //通过取地址拿到i的指针
	*pi = 44
	fmt.Println(i)
	fmt.Println(*pi)
	//也可以简单点, reflect.ValueOf(&i)就是i的指针
	pi = reflect.ValueOf(&i).Interface().(*int)
	*pi = 55
	fmt.Println(i)
}

//函数调用
func Test10(t *testing.T) {
	var foo = &Foo{"abc", 10}
	rv := reflect.ValueOf(foo)
	rv.MethodByName("Do").Call([]reflect.Value{})
	var n int = 5
	params := []reflect.Value{reflect.ValueOf(n)}
	rv.MethodByName("Sum").Call(params)
	_, ok := rv.Type().MethodByName("Sum")
	if !ok {
		t.Error("no method matched")
	}
	method := rv.MethodByName("Sum")
	rsp := method.Call(params)
	result := rsp[0].Interface().(int)
	fmt.Println(result)

}
