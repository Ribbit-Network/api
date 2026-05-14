package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
)

func runKeygen(args []string) {
	if len(args) < 1 {
		keygenUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "issue":
		keygenIssue(args[1:])
	case "list":
		keygenList(args[1:])
	case "revoke":
		keygenRevoke(args[1:])
	default:
		keygenUsage()
		os.Exit(2)
	}
}

func keygenUsage() {
	fmt.Fprintln(os.Stderr, "usage: api keygen <issue|list|revoke> [flags]")
}

func keygenIssue(args []string) {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	owner := fs.String("owner", "", "owner label (e.g. an email or team name)")
	_ = fs.Parse(args)

	if *owner == "" {
		log.Fatal("--owner is required")
	}
	store, err := openKeyStore()
	if err != nil {
		log.Fatal(err)
	}

	raw, k, err := store.Issue(*owner)
	if err != nil {
		log.Fatalf("issue: %v", err)
	}

	fmt.Println("API key issued. Store it now — it will not be shown again.")
	fmt.Printf("  id:    %d\n", k.ID)
	fmt.Printf("  owner: %s\n", k.Owner)
	fmt.Printf("  key:   %s\n", raw)
}

func keygenList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	_ = fs.Parse(args)

	store, err := openKeyStore()
	if err != nil {
		log.Fatal(err)
	}
	keys, err := store.List()
	if err != nil {
		log.Fatalf("list: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPREFIX\tOWNER\tCREATED\tREVOKED")
	for _, k := range keys {
		revoked := "-"
		if k.RevokedAt != nil {
			revoked = k.RevokedAt.Format("2006-01-02")
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			k.ID, k.Prefix, k.Owner, k.CreatedAt.Format("2006-01-02"), revoked)
	}
	_ = w.Flush()
}

func keygenRevoke(args []string) {
	fs := flag.NewFlagSet("revoke", flag.ExitOnError)
	id := fs.Int64("id", 0, "key id to revoke")
	_ = fs.Parse(args)

	if *id == 0 {
		log.Fatal("--id is required")
	}
	store, err := openKeyStore()
	if err != nil {
		log.Fatal(err)
	}
	if err := store.Revoke(*id); err != nil {
		log.Fatalf("revoke: %v", err)
	}
	fmt.Printf("revoked key id=%d\n", *id)
}
