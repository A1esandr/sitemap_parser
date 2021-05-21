# sitemap_parser
Tool for parsing XML sitemap.
* Creates list of all site pages. Prints list to console by defaults. 
* Can execute backup of loaded pages if backup path provided.

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
Envs have precedence over command line args
* **SITE** - (Required) URL of site with sitemal.xml, for example, https://alextech18.blogspot.com
* **BACKUP_PATH** - (Optional) path for backuping loaded website pages, for example, /home/A1esandr/backups

#### Command line args
Command line args analized if envs are not present
* **-site** - (Required, or existence of SITE env) URL of site with sitemal.xml, for example, 
```
go run parser.go -site https://alextech18.blogspot.com
```
* **-backup** - (Optional) path for backuping loaded website pages, for example, 
```
go run parser.go -site https://alextech18.blogspot.com -backup /home/A1esandr/backups
```
