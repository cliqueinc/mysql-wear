package main

// import (
// 	"errors"
// 	"fmt"
// 	"os"
// 	"strings"
// )

// /*
// How will be used:
// - Install this binary
// - This bin is looking in the project home for a configurable migrations folder like
//  schema_updates/1,2,3,4,5
// */

// func help() {
// 	fmt.Println(`Basic Commands: "$ pgccmd up|init|rollback|status"`)
// 	fmt.Println(`Generate init: "$ pgccmd gen init StructName shortName"`)
// 	fmt.Println("")
// 	fmt.Println("For commands other than gen, pass -d for debug query logging")
// 	os.Exit(-1)
// }

// // RunCmd runs pgc commands with args.
// // pgc is usually bind to some environment and pgc migrations
// func RunCmd(args ...string) {
// 	if len(args) < 2 {
// 		help()
// 	}

// 	action := args[1]
// 	arg2 := ""

// 	if len(args) > 2 {
// 		arg2 = args[2]
// 		if arg2 == "-d" {
// 			// GetConfig().LogQueries = true
// 			if len(args) > 3 && !strings.HasPrefix(args[3], "-") {
// 				arg2 = args[3]
// 			}
// 		}
// 	}
// 	var err error

// 	switch action {
// 	case "up":
// 		execDefault := arg2 == "--exec-default"
// 		err = UpdateSchema(execDefault)
// 	case "migration":
// 		isDefault := len(args) > 2 && args[2] == "default"
// 		err = InitMigration(isDefault)
// 	case "exec":
// 		if arg2 == "" {
// 			err = errors.New("please specify migration name to execute")
// 			break
// 		}
// 		err = ExecuteMigration(arg2)
// 	case "init":
// 		// TODO do we want to allow an additional flag for initing with default?
// 		err = InitSchema(false)
// 	case "reset":
// 		err = reset()
// 	case "gen":
// 		err = Generate(args)
// 	case "rollback":
// 		if arg2 != "" && !strings.HasPrefix(arg2, "-") {
// 			err = Rollback(arg2)
// 		} else {
// 			err = RollbackLatest()
// 		}
// 	case "status":
// 		err = PrintVersionStatus()
// 	case "help":
// 		help()
// 	default:
// 		help()
// 	}

// 	if err != nil {
// 		fmt.Printf("Failed: %v\n", err)
// 		os.Exit(-1)
// 	}
// 	os.Exit(0)
// }

// /*
// See readme for details on generation
// */
// func Generate(args []string) error {
// 	// Right now we only support `pgccmd gen init StructName`
// 	if len(args) != 5 || args[2] != "init" {
// 		return errors.New("gen command currently only supports `$ pgccmd gen init StructName shortName`")
// 	}
// 	fmt.Println(GenerateInit(args[3], args[4]))
// 	return nil
// }
