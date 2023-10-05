package debefix

type ResolveValue interface {
	isResolveValue()
}

type ResolveGenerate struct {
}

func (r ResolveGenerate) isResolveValue() {}

type ResolveContext interface {
	TableID() string
	TableName() string
	ResolveField(fieldName string, value any)
}

type defaultResolveContext struct {
	tableID, tableName string
	resolved           map[string]any
}

func (d *defaultResolveContext) TableID() string {
	return d.tableID
}

func (d *defaultResolveContext) TableName() string {
	return d.tableName
}

func (d *defaultResolveContext) ResolveField(fieldName string, value any) {
	if d.resolved == nil {
		d.resolved = map[string]any{}
	}
	d.resolved[fieldName] = value
}
