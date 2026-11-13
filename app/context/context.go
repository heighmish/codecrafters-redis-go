package connContext

type Context struct {
	Id int
}

func NewContext(id int) *Context {
	return &Context{
		Id: id,
	}
}
