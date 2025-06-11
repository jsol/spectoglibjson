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

func to_func_name(title string) string {

	title = strings.ToLower(title)
	title = strings.Replace(title, " ", "_", -1)

	return "create_" + title
}

func to_c_name(name string) string {
	var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

	snake := matchFirstCap.ReplaceAllString(name, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)

}

func printGetFunc(name string, properties map[string]Property, required []string) {

	var asserts string
	var toJson []string
	var toJsonOpt []string
	var args []string
	var args_opt []string
	var docParam []string
	var localParams []string
	var freeParams []string

	localParams = append(localParams, "JsonObject *obj = NULL;")
	for name, prop := range properties {
		cname := to_c_name(name)

		if prop.Type == "object" {

			printGetFunc(name, prop.Properties, prop.Required)
			docParam = append(docParam, "* @param "+cname+"  "+prop.Description)
			toJsonStr := "json_object_set_object_member(obj, \"" + name + "\", " + cname + ");"
			if slices.Contains(required, name) {
				args = append(args, "JsonObject * "+cname)
				toJson = append(toJson, toJsonStr)
				asserts += "\ng_assert(" + cname + ");"
			} else {
				args_opt = append(args_opt, "JsonObject * "+cname)
				toJsonOpt = append(toJsonOpt, "if ("+cname+" != NULL) {\n"+toJsonStr+"\n}\n")
			}

		} else if prop.Type == "boolean" {
			docParam = append(docParam, "* @param "+cname+"  "+prop.Description)
			toJsonStr := "json_object_set_boolean_member(obj, \"" + name + "\", " + cname + ");"

			if slices.Contains(required, name) {
				args = append(args, "gboolean "+to_c_name(name))
			} else {
				args_opt = append(args_opt, "gboolean "+to_c_name(name))
			}
			toJson = append(toJson, toJsonStr)
		} else if prop.Type == "string" {

			if prop.Format == "date-time" {
				docParam = append(docParam, "* @param "+cname+"  "+prop.Description)
				localParams = append(localParams, "gchar *"+cname+"_str = NULL;")
				toJsonStr := cname + "_str = bwc_utils_dt_fmt_to_rfc3339(" + cname + ");\n"
				toJsonStr += "json_object_set_string_member(obj, \"" + name + "\", " + cname + "_str);"
				freeParams = append(freeParams, "g_free("+cname+"_str);")

				if slices.Contains(required, name) {
					args = append(args, "GDateTime * "+cname)
					toJson = append(toJson, toJsonStr)
					asserts += "\ng_assert(" + cname + ");"
				} else {
					args_opt = append(args_opt, "GDateTime * "+cname)
					toJsonOpt = append(toJsonOpt, "if ("+cname+" != NULL) {\n"+toJsonStr+"\n}\n")
				}

			} else {

				docParam = append(docParam, "* @param "+cname+"  "+prop.Description)
				toJsonStr := "json_object_set_string_member(obj, \"" + name + "\", " + cname + ");"

				if slices.Contains(required, name) {
					args = append(args, "const gchar * "+to_c_name(name))
					toJson = append(toJson, toJsonStr)
					asserts += "\ng_assert(" + to_c_name(name) + ");"
				} else {
					args_opt = append(args_opt, "const gchar * "+to_c_name(name))
					toJsonOpt = append(toJsonOpt, "if ("+to_c_name(name)+" != NULL) {\n"+toJsonStr+"\n}\n")
				}
			}
		} else {
			log.Fatal("Unknown type", prop.Type)
		}
	}

	output := "static JsonObject * " + to_func_name(name)
	output += "("
	output += strings.Join(args, ", ")
	if len(args_opt) > 0 {
		if len(args) > 0 {
			output += ", "
		}
		output += strings.Join(args_opt, ", ")
	}
	output += ") {\n"
	output += strings.Join(localParams, "\n")
	output += "\n\n"
	output += asserts
	output += "\n\n"
	output += "obj = json_object_new();\n\n"
	output += strings.Join(toJson, "\n")
	output += "\n\n"
	output += strings.Join(toJsonOpt, "\n")
	output += "\n\n"
	output += strings.Join(freeParams, "\n")
	output += "\n\n"
	output += "return obj;"
	output += "\n}"

	fmt.Println(output)

}

func main() {
	var meta Meta

	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], " <file.json>")
		os.Exit(1)
	}
	schema, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal("Failed to open JSON schema")
	}

	jsonParser := json.NewDecoder(schema)
	if err = jsonParser.Decode(&meta); err != nil {
		log.Fatal("parsing schema file", os.Args[1], err.Error())
	}

	printGetFunc(meta.Title, meta.Properties, meta.Required)
}
