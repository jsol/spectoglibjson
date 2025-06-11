package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"
)

type Property struct {
	Type        string              `json:"type"`
	Format      string              `json:"format"`
	Description string              `json:"description"`
	Examples    []string            `json:"examples"`
	Properties  map[string]Property `json:"properties"`
	Required    []string            `json:"required"`
}

type Meta struct {
	Schema               string              `json:"$schema"`
	ID                   string              `json:"$id"`
	Title                string              `json:"title"`
	Description          string              `json:"description"`
	Type                 string              `json:"type"`
	AdditionalProperties bool                `json:"additionalProperties"`
	Properties           map[string]Property `json:"properties"`
	Required             []string            `json:"required"`
}

type Output struct {
	Code   string
	Header string

	Namespace string
	Required  []Output
}

type GObjectProp struct {
	Name        string
	Enum        string
	CType       string
	Comment     string
	PSpec       string
	Setter      string
	Getter      string
	Constructor string
	Free        string
}

type GObject struct {
	Namespace    string
	Name         string
	ExtraHeaders []string
	Props        []GObjectProp
}

func to_func_name(title string) string {

	title = strings.ToLower(title)
	title = strings.Replace(title, " ", "_", -1)

	return title
}

func to_c_name(name string) string {
	var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

	snake := matchFirstCap.ReplaceAllString(name, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)

}

func toClassName(namespace, name string) string {
	return strings.ToUpper(namespace[:1]) + strings.ToLower(namespace[1:]) + strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
}
func toTypeName(namespace, name string) string {
	return strings.ToUpper(namespace) + "_TYPE_" + strings.ToUpper(name)
}

func generate(namespace, name string, properties map[string]Property, required []string) []GObject {

	out := []GObject{}
	obj := GObject{Namespace: namespace, Name: name, ExtraHeaders: []string{}, Props: []GObjectProp{}}

	for name, prop := range properties {
		cname := to_c_name(name)
		p := GObjectProp{Name: cname, Enum: "PROP_" + strings.ToUpper(to_c_name(obj.Name)) + "_" + strings.ToUpper(to_c_name(name))}
		p.Comment = prop.Description

		if prop.Type == "object" {
			out = append(out, generate(namespace, name, prop.Properties, prop.Required)...)
			p.CType = toClassName(namespace, name) + "*"

			p.PSpec = "g_param_spec_object(\"" + to_c_name(name) + "\", \"" + name + "\", " + prop.Description + "\", " + toTypeName(namespace, name) + ",G_PARAM_READWRITE | G_PARAM_EXPLICIT_NOTIFY);"

			p.Setter = "self->" + p.Name + " = g_value_get_object(value);"
			p.Getter = "g_value_set_object(value, self->" + p.Name + ");"

			if slices.Contains(required, name) {
				p.Constructor = p.CType + " " + cname
			}

			p.Free = "g_clear_object(&self->" + cname + ");"

		} else if prop.Type == "boolean" {
			p.CType = "gboolean"
			p.PSpec = "g_param_spec_boolean(\"" + to_c_name(name) + "\", \"" + name + "\", \"" + prop.Description + "\", FALSE, G_PARAM_READWRITE | G_PARAM_EXPLICIT_NOTIFY);"

			p.Setter = "self->" + p.Name + " = g_value_get_boolean(value);"
			p.Getter = "g_value_set_boolean(value, self->" + p.Name + ");"

			if slices.Contains(required, name) {
				p.Constructor = p.CType + " " + cname
			}
		} else if prop.Type == "integer" {
			p.CType = "gint64"
			p.PSpec = "g_param_spec_int64(\"" + to_c_name(name) + "\", \"" + name + "\", \"" + prop.Description + "\", G_MININT64, G_MAXINT64, 0, G_PARAM_READWRITE | G_PARAM_EXPLICIT_NOTIFY);"

			p.Setter = "self->" + p.Name + " = g_value_get_double(value);"
			p.Getter = "g_value_set_int64(value, self->" + p.Name + ");"

			if slices.Contains(required, name) {
				p.Constructor = p.CType + " " + cname
			}
		} else if prop.Type == "number" {
			p.CType = "gdouble"
			p.PSpec = "g_param_spec_double(\"" + to_c_name(name) + "\", \"" + name + "\", \"" + prop.Description + "\", -G_MAXDOUBLE, G_MAXDOUBLE, 0.0, G_PARAM_READWRITE | G_PARAM_EXPLICIT_NOTIFY);"

			p.Setter = "self->" + p.Name + " = g_value_get_double(value);"
			p.Getter = "g_value_set_double(value, self->" + p.Name + ");"

			if slices.Contains(required, name) {
				p.Constructor = p.CType + " " + cname
			}
		} else if prop.Type == "string" {
			p.CType = "gchar *"

			p.PSpec = "g_param_spec_string(\"" + to_c_name(name) + "\", \"" + name + "\", \"" + prop.Description + "\", NULL,G_PARAM_READWRITE | G_PARAM_EXPLICIT_NOTIFY);"

			p.Setter = "self->" + p.Name + " = g_value_get_string(value);"
			p.Getter = "g_value_set_string(value, self->" + p.Name + ");"

			if slices.Contains(required, name) {
				p.Constructor = p.CType + " " + cname
			}
			p.Free = "g_free(self->" + cname + ");"
		} else {
			log.Fatal("Unknown type", prop.Type)
		}
		obj.Props = append(obj.Props, p)
	}

	out = append(out, obj)
	return out
}

func generateHeader(obj GObject) string {
	fqdn := obj.Namespace + "_" + obj.Name
	out := "#define " + toTypeName(obj.Namespace, obj.Name) + " " + obj.Namespace + "_" + obj.Name + "_get_type()\n"
	out += "G_DECLARE_FINAL_TYPE(" + toClassName(obj.Namespace, obj.Name) + "," + obj.Namespace + "_" + obj.Name + ", " + strings.ToUpper(obj.Namespace) + ", " + strings.ToUpper(obj.Name) + ", GObject)\n\n"

	out += toClassName(obj.Namespace, obj.Name) + " * " + obj.Namespace + "_" + obj.Name + "_new("

	set := false

	for _, prop := range obj.Props {
		if prop.Constructor != "" {
			if set {
				out += ", "
			}
			out += prop.Constructor
			set = true
		}
	}
	if !set {
		out += "void"
	}
	out += ");"
	for _, prop := range obj.Props {
		out += "void " + fqdn + "_set_" + prop.Name + "(" + toClassName(obj.Namespace, obj.Name) + " *self, " + prop.CType + " " + prop.Name + ");\n\n"
	}

	return out
}

func generateCodePreamble(obj GObject) string {

	out := "struct _" + toClassName(obj.Namespace, obj.Name) + " {\n  GObject parent;\n"
	for _, prop := range obj.Props {
		out += prop.CType + " " + prop.Name + "; /**< " + prop.Comment + "*/"
	}

	out += "};\n\n"
	out += "G_DEFINE_TYPE(" + toClassName(obj.Namespace, obj.Name) + " , " + obj.Namespace + "_" + obj.Name + ", G_TYPE_OBJECT)\n\n"

	out += "typedef enum {"

	for i, prop := range obj.Props {
		if i == 0 {
			out += prop.Enum + " = 1,"
		} else {
			out += prop.Enum + ","
		}
	}

	out += " \n} " + toClassName(obj.Namespace, obj.Name) + "Property;\n\n"

	return out
}

func cast(obj GObject) string {
	return toClassName(obj.Namespace, obj.Name) + " *self = " + strings.ToUpper(obj.Namespace) + "_" + strings.ToUpper(obj.Name) + "(object);"

}
func generateCode(obj GObject) string {

	fqdn := obj.Namespace + "_" + obj.Name
	out := "static void\n" + fqdn + "_finalize(GObject *object)\n{"
	out += cast(obj)

	out += "\n  g_assert(self);"

	for _, prop := range obj.Props {
		out += prop.Free + "\n"
	}

	out += "  G_OBJECT_CLASS(" + fqdn + "_parent_class)->finalize(obj);"
	out += "}\n\n"

	out += "static void\n" + fqdn + "_get_property(GObject *object, guint property_id, GValue *value, GParamSpec *pspec) {\n"
	out += cast(obj)

	out += "\n\n   switch ((" + toClassName(obj.Namespace, obj.Name) + "Property) property_id) {\n"

	for _, prop := range obj.Props {
		out += "case " + prop.Enum + ":\n"
		out += prop.Getter
		out += "\n  break;\n"
	}

	out += `  default:
    G_OBJECT_WARN_INVALID_PROPERTY_ID(object, property_id, pspec);
    break;
  }
}
  `
	out += "static void\n" + fqdn + "_set_property(GObject *object, guint property_id, GValue *value, GParamSpec *pspec) {\n"
	out += cast(obj)

	out += "\n\n   switch ((" + toClassName(obj.Namespace, obj.Name) + "Property) property_id) {\n"

	for _, prop := range obj.Props {
		out += "case " + prop.Enum + ":\n"
		out += prop.Free
		out += prop.Setter
		out += "\n  break;\n"
	}

	out += `  default:
    G_OBJECT_WARN_INVALID_PROPERTY_ID(object, property_id, pspec);
    break;
  }
}
  `

	out += "static void\n" + fqdn + "_class_init(" + toClassName(obj.Namespace, obj.Name) + "Class *klass)\n{"
	out += "GParamSpec *pspec;\n\n"
	out += "   GObjectClass *object_class = G_OBJECT_CLASS(klass);"

	out += "  object_class->finalize = " + fqdn + "_finalize;"
	out += "  object_class->set_property = " + fqdn + "_set_property;"
	out += "  object_class->get_property = " + fqdn + "_get_property;"

	for _, prop := range obj.Props {
		out += "pspec = " + prop.PSpec
		out += "g_object_class_install_property(object_class, " + prop.Enum + ", pspec);\n"
		out += "g_param_spec_unref(pspec);\n"
	}

	out += "\n}\n\n"
	return out
}

func main() {
	var meta Meta

	if len(os.Args) != 4 {
		fmt.Println("Usage: ", os.Args[0], " <namespace> <main class name> <file.json>")
		os.Exit(1)
	}
	schema, err := os.Open(os.Args[3])
	if err != nil {
		log.Fatal("Failed to open JSON schema")
	}

	jsonParser := json.NewDecoder(schema)
	if err = jsonParser.Decode(&meta); err != nil {
		log.Fatal("parsing schema file", os.Args[3], err.Error())
	}

	obj := generate(os.Args[1], to_func_name(meta.Title), meta.Properties, meta.Required)

	headers := `#pragma once

#include <glib.h>
#include <glib-object.h>

G_BEGIN_DECLS
`

	code := `
  #include <glib.h>
  #include <glib-object.h>`

	code += "\n#include \"" + os.Args[1] + "_" + os.Args[2] + ".h\"\n"
	for _, info := range obj {
		headers += generateHeader(info)
		code += generateCodePreamble(info)
	}

	for _, info := range obj {
		code += generateCode(info)
	}
	headers += "\nG_END_DECLS"

  err = os.WriteFile(os.Args[1] + "_" + os.Args[2] + ".h",[]byte( headers), 0644)
	if err != nil {
		log.Panic(err)
	}

  err = os.WriteFile(os.Args[1] + "_" + os.Args[2] + ".c",[]byte( code), 0644)
	if err != nil {
		log.Panic(err)
	}

	for _, info := range obj {
		fmt.Println("OBJ:"+info.Name, len(info.Props))
	}
}
