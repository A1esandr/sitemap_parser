# sitemap_parser
Tool for parsing XML sitemap - to create list of all site pages. Prints list to console by defaults.
Expects site has sitemap.xml file, for example, https://alextech18.blogspot.com/sitemap.xml

### Prerequisites
* Go 1.16

### Usage

#### Loading dependencies
```
go mod tidy
```
#### Run
```
go run parser.go
```

### Settings

#### Envs
* **SITE** - URL of site with sitemal.xml, for example, https://alextech18.blogspot.com
* **BACKUP_PATH** - path for backuping loaded website pages, for example, /home/A1esandr/backups

#### Command line args
* **-site** - URL of site with sitemal.xml, for example, ```go run parser.go -site https://alextech18.blogspot.com```
* **-backup** - path for backuping loaded website pages, for example, ```go run parser.go -site https://alextech18.blogspot.com -backup /home/A1esandr/backups```
