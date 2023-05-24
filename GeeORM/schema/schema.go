package schema

import (
	"geeorm/dialect"
	"go/ast"
	"reflect"
)

// 列（字段）
type Field struct {
	// 列名
	Name string
	// 值类型
	Type string
	// 约束条件，如主键
	Tag string
}

// 表
type Schema struct {
	// 被映射的对象，即原对象
	Model interface{}
	// 表名
	Name string
	// 所有字段
	Fields []*Field
	// 所有字段名
	FiledNames []string
	// 字段名和 Field 的映射
	fieldMap map[string]*Field
}

func (schema *Schema) GetField(name string) *Field {
	return schema.fieldMap[name]
}

// 将对象解析成Schema实例
func Parse(dest interface{}, dia dialect.Dialect) *Schema {
	// 获取结构体值的所有属性的类型信息
	modelType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	schema := &Schema{
		Model:    dest,
		Name:     modelType.Name(),
		fieldMap: make(map[string]*Field),
	}

	for i := 0; i < modelType.NumField(); i++ {
		p := modelType.Field(i)

		// 属性显示存在并且首字母大写
		if !p.Anonymous && ast.IsExported(p.Name) {
			field := &Field{
				Name: p.Name,
				Type: dia.DataTypeOf(reflect.Indirect(reflect.New(p.Type))),
			}

			if tag, ok := p.Tag.Lookup("geeorm"); ok {
				field.Tag = tag
			}

			schema.Fields = append(schema.Fields, field)
			schema.FiledNames = append(schema.FiledNames, p.Name)
			schema.fieldMap[p.Name] = field
		}
	}

	return schema
}

// 根据数据库中列的顺序，从对象中找到对应的值，按顺序平铺
func (schema *Schema) RecordValues(dest interface{}) []interface{} {
	destValue := reflect.Indirect(reflect.ValueOf(dest))
	var fieldValues []interface{}

	for _, field := range schema.Fields {
		fieldValues = append(fieldValues, destValue.FieldByName(field.Name).Interface())
	}

	return fieldValues
}
