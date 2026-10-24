package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/modules/task/repo"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
	"github.com/programme-lv/backend/modules/task/storage"
)

func runMaintenance(name string, args []string) error {
	p, pos, err := parseFlags(spec(name), args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandHelp(os.Stdout, spec(name))
		return nil
	}
	if err != nil {
		return err
	}
	if (name == "rename" && len(pos) != 2) || (name == "storage-cleanup" && len(pos) != 0) {
		return fmt.Errorf("%w: invalid arguments for %s", errUsage, name)
	}
	if name == "rename" && (!storage.ValidID(pos[0]) || !storage.ValidID(pos[1]) || pos[0] == pos[1]) {
		return fmt.Errorf("%w: expected two distinct valid task IDs", errUsage)
	}
	root := conf.MustGetFileStorageRootFromEnv()
	lock, err := storage.Lock(root)
	if err != nil {
		return err
	}
	defer lock.Close()
	pool, err := conf.GetPgxPoolFromEnv()
	if err != nil {
		return err
	}
	defer pool.Close()
	dbUnlock, err := storage.LockDatabase(context.Background(), pool)
	if err != nil {
		return err
	}
	defer dbUnlock()
	conf.MustRunPostgresMigrationsFromEnv()
	m := storage.Maintenance{DB: pool, Root: root}
	ctx := context.Background()
	if err := m.Prepare(ctx); err != nil {
		return err
	}
	public := conf.MustGetFileStore(root, "")
	tests := conf.MustGetFileStore(root, "testfiles")
	tasksrvc.NewTaskSrvc(repo.NewTaskPgRepo(pool), public, tests).MigrateLegacyArchiveZips(ctx)
	if name == "rename" {
		if err := m.Rename(ctx, pos[0], pos[1]); err != nil {
			return err
		}
		fmt.Printf("Renamed %s -> %s; old links remain valid.\nUpdate your local TaskZip directory name and task.toml ID, or export %s.\n", pos[0], pos[1], pos[1])
		return nil
	}
	files, err := m.Cleanup(ctx, p.bools["apply"])
	if err != nil {
		return err
	}
	for _, name := range files {
		fmt.Println(name)
	}
	if p.bools["apply"] {
		fmt.Printf("Moved %d unreferenced public files into %s/recovery/.\n", len(files), root)
	} else {
		fmt.Printf("%d unreferenced public files; use --apply to move them into recovery/.\n", len(files))
	}
	return nil
}
