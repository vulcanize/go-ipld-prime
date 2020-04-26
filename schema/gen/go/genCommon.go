package gengo

import (
	"io"
)

/*
	This file is full of "typical" templates.
	They may not be used by *every* type and representation,
	but if they're extracted here, they're at least used by *many*.
*/

// emitNativeMaybe turns out to be completely agnostic to pretty much everything;
// it doesn't vary by kind at all, and has never yet ended up needing specialization.
func emitNativeMaybe(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Maybe struct {
			m schema.Maybe
			v {{if not (MaybeUsesPtr .Type) }}_{{end}}{{ .Type | TypeSymbol }}
		}
		type Maybe{{ .Type | TypeSymbol }} = *_{{ .Type | TypeSymbol }}__Maybe

		func (m Maybe{{ .Type | TypeSymbol }}) IsNull() bool {
			return m.m == schema.Maybe_Null
		}
		func (m Maybe{{ .Type | TypeSymbol }}) IsUndefined() bool {
			return m.m == schema.Maybe_Absent
		}
		func (m Maybe{{ .Type | TypeSymbol }}) Exists() bool {
			return m.m == schema.Maybe_Value
		}
		func (m Maybe{{ .Type | TypeSymbol }}) AsNode() ipld.Node {
			switch m.m {
				case schema.Maybe_Absent:
					return ipld.Undef
				case schema.Maybe_Null:
					return ipld.Null
				case schema.Maybe_Value:
					return {{if not (MaybeUsesPtr .Type) }}&{{end}}m.v
				default:
					panic("unreachable")
			}
		}
		func (m Maybe{{ .Type | TypeSymbol }}) Must() {{ .Type | TypeSymbol }} {
			if !m.Exists() {
				panic("unbox of a maybe rejected")
			}
			return {{if not (MaybeUsesPtr .Type) }}&{{end}}m.v
		}
	`, w, adjCfg, data)
}

func emitNativeType_scalar(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// Using a struct with a single member is the same size in memory as a typedef,
	//  while also having the advantage of meaning we can block direct casting,
	//   which is desirable because the compiler then ensures our validate methods can't be evaded.
	doTemplate(`
		type _{{ .Type | TypeSymbol }} struct{ x {{ .ReprKind | KindPrim }} }
		type {{ .Type | TypeSymbol }} = *_{{ .Type | TypeSymbol }}
	`, w, adjCfg, data)
}

func emitNativeAccessors_scalar(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// The node interface's `AsFoo` method is almost sufficient... but
	//  this method unboxes without needing to return an error that's statically impossible,
	//   which makes it easier to use in chaining.
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) {{ .ReprKind.String | title }}() {{ .ReprKind | KindPrim }} {
			return n.x
		}
	`, w, adjCfg, data)
}

func emitNativeBuilder_scalar(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// Generate a single-step construction function -- this is easy to do for a scalar,
	//  and all representations of scalar kind can be expected to have a method like this.
	// The function is attached to the nodestyle for convenient namespacing;
	//  it needs no new memory, so it would be inappropriate to attach to the builder or assembler.
	// FUTURE: should engage validation flow.
	doTemplate(`
		func (_{{ .Type | TypeSymbol }}__Style) From{{ .ReprKind.String | title }}(v {{ .ReprKind | KindPrim }}) ({{ .Type | TypeSymbol }}, error) {
			n := _{{ .Type | TypeSymbol }}{v}
			return &n, nil
		}
	`, w, adjCfg, data)
}

func emitNodeTypeAssertions_typical(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		var _ ipld.Node = ({{ .Type | TypeSymbol }})(&_{{ .Type | TypeSymbol }}{})
		var _ schema.TypedNode = ({{ .Type | TypeSymbol }})(&_{{ .Type | TypeSymbol }}{})
	`, w, adjCfg, data)
}

func emitNodeMethodAsKind_scalar(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) As{{ .ReprKind.String | title }}() ({{ .ReprKind | KindPrim }}, error) {
			return n.x, nil
		}
	`, w, adjCfg, data)
}

func emitNodeMethodStyle_typical(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Style() ipld.NodeStyle {
			return _{{ .Type | TypeSymbol }}__Style{}
		}
	`, w, adjCfg, data)
}

// nodeStyle doesn't really vary textually at all between types and kinds
// because it's just builders and standard resets.
func emitNodeStyleType_typical(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Style struct{}

		func (_{{ .Type | TypeSymbol }}__Style) NewBuilder() ipld.NodeBuilder {
			var nb _{{ .Type | TypeSymbol }}__Builder
			nb.Reset()
			return &nb
		}
	`, w, adjCfg, data)
}

// emitTypicalTypedNodeMethodRepresentation does... what it says on the tin.
//
// For most types, the way to get the representation node pointer doesn't
// textually depend on either the node implementation details nor what the representation strategy is,
// or really much at all for that matter.
// It only depends on that they have the same structure, so this cast works.
//
// Most (all?) types can use this.  However, it's here rather in the mixins, for two reasons:
// one, it still seems possible to imagine we'll have a type someday for which this pattern won't hold;
// and two, mixins are also used in the repr generators, and it wouldn't be all sane for this method to end up also on reprs.
func emitTypicalTypedNodeMethodRepresentation(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) Representation() ipld.Node {
			return (*_{{ .Type | TypeSymbol }}__Repr)(n)
		}
	`, w, adjCfg, data)
}

// Turns out basically all builders are just an embed of the corresponding assembler.
func emitEmitNodeBuilderType_typical(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Builder struct {
			_{{ .Type | TypeSymbol }}__Assembler
		}
	`, w, adjCfg, data)
}

// Builder build and reset methods are common even when some parts of the assembler vary.
// We count on the zero value of any addntl non-common fields of the assembler being correct.
func emitNodeBuilderMethods_typical(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		func (nb *_{{ .Type | TypeSymbol }}__Builder) Build() ipld.Node {
			if *nb.m != schema.Maybe_Value {
				panic("invalid state: cannot call Build on an assembler that's not finished")
			}
			return nb.w
		}
		func (nb *_{{ .Type | TypeSymbol }}__Builder) Reset() {
			var w _{{ .Type | TypeSymbol }}
			var m schema.Maybe
			*nb = _{{ .Type | TypeSymbol }}__Builder{_{{ .Type | TypeSymbol }}__Assembler{w: &w, m: &m}}
		}
	`, w, adjCfg, data)
}

// emitNodeAssemblerType_scalar emits a NodeAssembler that's typical for a scalar.
// Types that are recursive tend to have more state and custom stuff, so won't use this
// (although the 'm' and 'w' variable names may still be presumed universally).
func emitNodeAssemblerType_scalar(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Assembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
		}

		func (na *_{{ .Type | TypeSymbol }}__Assembler) reset() {}
	`, w, adjCfg, data)
}

func emitNodeAssemblerMethodAssignNull_scalar(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) AssignNull() error {
			switch *na.m {
			case allowNull:
				*na.m = schema.Maybe_Null
				return nil
			case schema.Maybe_Absent:
				return mixins.{{ .ReprKind.String | title }}Assembler{"{{ .PkgName }}.{{ .TypeName }}{{ if .IsRepr }}.Repr{{end}}"}.AssignNull()
			case schema.Maybe_Value, schema.Maybe_Null:
				panic("invalid state: cannot assign into assembler that's already finished")
			}
			panic("unreachable")
		}
	`, w, adjCfg, data)
}

// almost the same as the variant for scalars, but also has to check for midvalue state.
func emitNodeAssemblerMethodAssignNull_recursive(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) AssignNull() error {
			switch *na.m {
			case allowNull:
				*na.m = schema.Maybe_Null
				return nil
			case schema.Maybe_Absent:
				return mixins.{{ .ReprKind.String | title }}Assembler{"{{ .PkgName }}.{{ .TypeName }}{{ if .IsRepr }}.Repr{{end}}"}.AssignNull()
			case schema.Maybe_Value, schema.Maybe_Null:
				panic("invalid state: cannot assign into assembler that's already finished")
			case midvalue:
				panic("invalid state: cannot assign null into an assembler that's already begun working on recursive structures!")
			}
			panic("unreachable")
		}
	`, w, adjCfg, data)
}

// works for the AssignFoo methods for scalar kinds that are just boxing a thing.
// There's no equivalent of this at all for recursives -- they're too diverse.
func emitNodeAssemblerMethodAssignKind_scalar(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// This method contains a branch to support MaybeUsesPtr because new memory may need to be allocated.
	//  This allocation only happens if the 'w' ptr is nil, which means we're being used on a Maybe;
	//  otherwise, the 'w' ptr should already be set, and we fill that memory location without allocating, as usual.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__Assembler) Assign{{ .ReprKind.String | title }}(v {{ .ReprKind | KindPrim }}) error {
			switch *na.m {
			case schema.Maybe_Value, schema.Maybe_Null:
				panic("invalid state: cannot assign into assembler that's already finished")
			}
			{{- if .Type | MaybeUsesPtr }}
			if na.w == nil {
				na.w = &_{{ .Type | TypeSymbol }}{}
			}
			{{- end}}
			na.w.x = v
			*na.m = schema.Maybe_Value
			return nil
		}
	`, w, adjCfg, data)
}

// leans heavily on the fact all the AsFoo and AssignFoo methods follow a very consistent textual pattern.
// FUTURE: may be able to get this to work for recursives, too -- but maps and lists each have very unique bottom thirds of this function.
func emitNodeAssemblerMethodAssignNode_scalar(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// AssignNode goes through three phases:
	// 1. is it null?  Jump over to AssignNull (which may or may not reject it).
	// 2. is it our own type?  Handle specially -- we might be able to do efficient things.
	// 3. is it the right kind to morph into us?  Do so.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__Assembler) AssignNode(v ipld.Node) error {
			if v.IsNull() {
				return na.AssignNull()
			}
			if v2, ok := v.(*_{{ .Type | TypeSymbol }}); ok {
				switch *na.m {
				case schema.Maybe_Value, schema.Maybe_Null:
					panic("invalid state: cannot assign into assembler that's already finished")
				}
				{{- if .Type | MaybeUsesPtr }}
				if na.w == nil {
					na.w = v2
					*na.m = schema.Maybe_Value
					return nil
				}
				{{- end}}
				*na.w = *v2
				*na.m = schema.Maybe_Value
				return nil
			}
			if v2, err := v.As{{ .ReprKind.String | title }}(); err != nil {
				return err
			} else {
				return na.Assign{{ .ReprKind.String | title }}(v2)
			}
		}
	`, w, adjCfg, data)
}
