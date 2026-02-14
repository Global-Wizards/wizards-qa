package main

import "time"

const (
	ModeAgent   = "agent"
	ModeBrowser = "browser"
	ModeMaestro = "maestro"

	AnalysisTimeout      = 15 * time.Minute
	TestExecutionTimeout = 10 * time.Minute
)
