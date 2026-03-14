package constitution_test

import (
	"context"
	"fmt"
	"log"

	"go-wind-admin/pkg/constitution"
)

func ExampleDocumentationSyncer_SyncAPIDocumentation() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Create documentation syncer
	syncer := constitution.NewDocumentationSyncer(cfg)

	// Sync API documentation from a proto file
	result, err := syncer.SyncAPIDocumentation(
		context.Background(),
		"backend/api/protos/identity/service/v1/user.proto",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Documentation generated: %s\n", result.FilePath)
	fmt.Printf("Changed: %v\n", result.Changed)
	if result.Changed {
		fmt.Printf("Changes: %s\n", result.ChangesSummary)
	}

	// Output:
	// Documentation generated: docs/api/userservice.md
	// Changed: true
	// Changes: Added 50 lines, removed 0 lines, modified 0 sections
}

func ExampleDocumentationSyncer_SyncComponentDocumentation() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Create documentation syncer
	syncer := constitution.NewDocumentationSyncer(cfg)

	// Sync component documentation from a Vue file
	result, err := syncer.SyncComponentDocumentation(
		context.Background(),
		"frontend/apps/admin/src/components/UserList.vue",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Documentation generated: %s\n", result.FilePath)
	fmt.Printf("Source link: %s\n", result.SourceLink)

	// Output:
	// Documentation generated: docs/components/userlist.md
	// Source link: https://github.com/your-org/your-repo/blob/main/frontend/apps/admin/src/components/UserList.vue
}

func ExampleDocumentationSyncer_ValidateDocumentation() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Create documentation syncer
	syncer := constitution.NewDocumentationSyncer(cfg)

	// Validate documentation completeness
	report, err := syncer.ValidateDocumentation(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total APIs: %d\n", report.TotalAPIs)
	fmt.Printf("Documented APIs: %d\n", report.DocumentedAPIs)
	fmt.Printf("Coverage: %.1f%%\n", report.CoveragePercent)
	fmt.Printf("Missing docs: %d\n", len(report.MissingDocs))

	// Output:
	// Total APIs: 10
	// Documented APIs: 8
	// Coverage: 80.0%
	// Missing docs: 2
}

func ExampleDocumentationSyncer_SearchDocumentation() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Create documentation syncer
	syncer := constitution.NewDocumentationSyncer(cfg)

	// Build search index first
	if err := syncer.BuildSearchIndex(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Search documentation
	results, err := syncer.SearchDocumentation(context.Background(), "user authentication")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d results\n", len(results))
	for i, result := range results {
		if i >= 3 {
			break
		}
		fmt.Printf("%d. %s (score: %.1f)\n", i+1, result.Title, result.Score)
	}

	// Output:
	// Found 5 results
	// 1. UserService (score: 25.0)
	// 2. Authentication Guide (score: 18.0)
	// 3. User Management (score: 12.0)
}

func ExampleDocumentationSyncer_GetDocumentationVersion() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Create documentation syncer
	syncer := constitution.NewDocumentationSyncer(cfg)

	// Get latest version
	version, err := syncer.GetDocumentationVersion(
		context.Background(),
		"docs/api/userservice.md",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Version: %s\n", version.Version)
	fmt.Printf("Created: %s\n", version.CreatedAt.Format("2006-01-02"))
	fmt.Printf("Author: %s\n", version.Author)

	// Output:
	// Version: v20260312-143000
	// Created: 2026-03-12
	// Author: constitution-syncer
}

func ExampleDocumentationSyncer_GenerateAPIReference() {
	// Load configuration
	cfg, err := constitution.LoadConfig(".ai/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Create documentation syncer
	syncer := constitution.NewDocumentationSyncer(cfg)

	// Generate complete API reference (processes all proto files concurrently)
	if err := syncer.GenerateAPIReference(context.Background(), "docs/api"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("API reference generated successfully")

	// Output:
	// API reference generated successfully
}
