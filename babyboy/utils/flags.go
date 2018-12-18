package utils

import (
	"github.com/babyboy/babyboy/frontsection"
	"github.com/babyboy/babyboy/log"
	"github.com/babyboy/babyboy/node"
	"github.com/babyboy/babyboy/urfave/cli"
	"os"
	"path/filepath"
)

var (
	CommandHelpTemplate = `{{.cmd.Name}}{{if .cmd.Subcommands}} command{{end}}{{if .cmd.Flags}} [command options]{{end}} [arguments...]
{{if .cmd.Description}}{{.cmd.Description}}
{{end}}{{if .cmd.Subcommands}}
SUBCOMMANDS:
	{{range .cmd.Subcommands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
	{{end}}{{end}}{{if .categorizedFlags}}
{{range $idx, $categorized := .categorizedFlags}}{{$categorized.Name}} OPTIONS:
{{range $categorized.Flags}}{{"\t"}}{{.}}
{{end}}
{{end}}{{end}}`
)

func init() {
	cli.AppHelpTemplate = `{{.Name}} {{if .Flags}}[global options] {{end}}command{{if .Flags}} [command options]{{end}} [arguments...]

VERSION:
   {{.Version}}

COMMANDS:
   {{range .Commands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
   {{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`

	cli.CommandHelpTemplate = CommandHelpTemplate
}

// NewApp creates an app with sane defaults.
func NewApp(gitCommit, usage string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = "BabyBoy Team"
	app.Email = ""
	if len(gitCommit) >= 8 {
		app.Version += "-" + gitCommit[:8]
	}
	app.Usage = usage
	return app
}

var (
	// General settings
	DataDirFlag = DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: DirectoryString{node.DefaultDataDir()},
	}
	NoDiscoverFlag = cli.BoolFlag{
		Name:  "nodiscover",
		Usage: "Disables the peer discovery mechanism (manual peer addition)",
	}
	P2pPortFlag = cli.IntFlag{
		Name:  "p2pport",
		Usage: "p2p port for communication",
	}
	RpcPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "rpc port for http server",
	}
	DbDirFlag = cli.IntFlag{
		Name:  "dbdir",
		Usage: "",
	}
)

// RegisterEthService adds an BabyBoy client to the stack.
func RegisterBabyService(stack *node.Node) {
	serviceCtx := GetServiceContext()
	if err := stack.Register(serviceCtx); err != nil {
		log.Error("Failed to register the BabyBoy services: %v", err)
	}
}

func GetServiceContext() node.ServiceConstructor {
	return func(ctx *node.ServiceContext) (node.Service, error) {
		fullNode, err := frontsection.New(ctx)

		return fullNode, err
	}
}
