package main

type ImageMigration struct {
	MigrationUnits []*MigrationUnit `yaml:"migrationUnits"`
}

type MigrationUnit struct {
	SourceImage      string `yaml:"sourceImage"`
	DestinationImage string `yaml:"destinationImage"`
}
