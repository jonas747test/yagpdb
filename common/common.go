package common

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/fzzy/radix/extra/pool"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/jonas747/discordgo"
)

const (
	VERSIONMAJOR = 0
	VERSIONMINOR = 19
	VERSIONPATCH = 3

	Testing = false // Disables stuff like command cooldowns
	// Testing = true // Disables stuff like command cooldowns
)

var (
	VERSIONNUMBER = fmt.Sprintf("%d.%d.%d", VERSIONMAJOR, VERSIONMINOR, VERSIONPATCH)
	VERSION       = VERSIONNUMBER + " Hygienic"

	SQL       *gorm.DB
	RedisPool *pool.Pool

	BotSession *discordgo.Session
	Conf       *CoreConfig

	AllPlugins []Plugin
)

func AddPlugin(p Plugin) {
	if AllPlugins == nil {
		AllPlugins = []Plugin{p}
		return
	}
	// Check for dupes
	for _, v := range AllPlugins {
		if v == p {
			return
		}
	}
	AllPlugins = append(AllPlugins, p)
}

type Plugin interface {
	Name() string
}

// Initalizes all database connections, config loading and so on
func Init() error {

	if Testing {
		logrus.SetLevel(logrus.DebugLevel)
	}

	config, err := LoadConfig()
	if err != nil {
		return err
	}
	Conf = config

	BotSession, err = discordgo.New(config.BotToken)
	if err != nil {
		return err
	}
	BotSession.MaxRestRetries = 3

	err = connectRedis(config.Redis)
	if err != nil {
		return err
	}

	err = connectDB(config.PQUsername, config.PQPassword)
	return err
}

func connectRedis(addr string) (err error) {
	RedisPool, err = pool.NewCustomPool("tcp", addr, 25, RedisDialFunc)
	if err != nil {
		logrus.WithError(err).Fatal("Failed initilizing redis pool")
	}
	return
}

func connectDB(user, pass string) error {
	db, err := gorm.Open("postgres", fmt.Sprintf("host=localhost user=%s dbname=yagpdb sslmode=disable password=%s", user, pass))
	SQL = db
	if err == nil {
		db.DB().SetMaxOpenConns(5)
	}

	return err
}
