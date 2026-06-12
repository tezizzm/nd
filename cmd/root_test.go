package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveVaultDir_PrefersSharedVaultForConfiguredWorktree(t *testing.T) {
	projectRoot, sharedVault := setupSharedWorktree(t)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	oldVault := vaultDir
	vaultDir = ""
	defer func() { vaultDir = oldVault }()

	if got := resolveVaultDir(); got != sharedVault {
		t.Fatalf("resolveVaultDir() = %q, want %q", got, sharedVault)
	}
}

func TestResolveVaultDir_FallsBackToNearestLocalVault(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "repo")
	nested := filepath.Join(projectRoot, "pkg", "service")

	if err := os.MkdirAll(filepath.Join(projectRoot, ".vault"), 0o755); err != nil {
		t.Fatal(err)
	}
	// resolveLocalVaultDir requires .nd.yaml to exist (not just the directory)
	if err := os.WriteFile(filepath.Join(projectRoot, ".vault", ".nd.yaml"), []byte("version: \"1\"\nprefix: TEST\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	oldVault := vaultDir
	vaultDir = ""
	defer func() { vaultDir = oldVault }()

	want := filepath.Join(projectRoot, ".vault")
	got := resolveVaultDir()
	if resolved, err := filepath.EvalSymlinks(got); err == nil {
		got = resolved
	}
	if resolved, err := filepath.EvalSymlinks(want); err == nil {
		want = resolved
	}
	if got != want {
		t.Fatalf("resolveVaultDir() = %q, want %q", got, want)
	}
}

func TestResolveVaultDir_UsesEnvironmentOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "override-vault")
	if err := os.Setenv("ND_VAULT_DIR", override); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("ND_VAULT_DIR") }()

	oldVault := vaultDir
	vaultDir = ""
	defer func() { vaultDir = oldVault }()

	if got := resolveVaultDir(); got != override {
		t.Fatalf("resolveVaultDir() = %q, want %q", got, override)
	}
}

// TestResolveVaultDir_WorktreeFallbackFindsMainVault verifies that when
// running inside a secondary git worktree where .vault is gitignored (absent),
// resolveVaultDir falls back to the main worktree's .vault.
func TestResolveVaultDir_WorktreeFallbackFindsMainVault(t *testing.T) {
	base := t.TempDir()

	// Main repo with .vault
	mainRepo := filepath.Join(base, "main-repo")
	mainGitDir := filepath.Join(mainRepo, ".git")
	mainVault := filepath.Join(mainRepo, ".vault")
	if err := os.MkdirAll(mainGitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(mainVault, 0o755); err != nil {
		t.Fatal(err)
	}

	// Secondary worktree -- NO .vault (gitignored), .git is a file
	worktree := filepath.Join(mainRepo, ".claude", "worktrees", "agent-abc")
	worktreeGitDir := filepath.Join(mainGitDir, "worktrees", "agent-abc")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(worktreeGitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Worktree .git file points to the worktree git dir
	gitPtr := "gitdir: " + filepath.ToSlash(worktreeGitDir) + "\n"
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte(gitPtr), 0o644); err != nil {
		t.Fatal(err)
	}
	// commondir points back to the main .git
	if err := os.WriteFile(filepath.Join(worktreeGitDir, "commondir"), []byte("../../\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(worktree); err != nil {
		t.Fatal(err)
	}

	oldVault := vaultDir
	vaultDir = ""
	defer func() { vaultDir = oldVault }()

	got := resolveVaultDir()
	if resolved, rerr := filepath.EvalSymlinks(got); rerr == nil {
		got = resolved
	}
	want := mainVault
	if resolved, rerr := filepath.EvalSymlinks(want); rerr == nil {
		want = resolved
	}
	if got != want {
		t.Fatalf("resolveVaultDir() from worktree = %q, want main vault %q", got, want)
	}
}

func TestResolveVaultDir_SiblingWorktreeFindsMainCheckoutConfig(t *testing.T) {
	base := t.TempDir()

	// Main repo with a real .git dir and the shared config in its checkout.
	mainRepo := filepath.Join(base, "main-repo")
	mainGitDir := filepath.Join(mainRepo, ".git")
	if err := os.MkdirAll(filepath.Join(mainRepo, ".vault"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "# nd shared-worktree state\nmode: git_common_dir\npath: paivot/nd-vault\n"
	if err := os.WriteFile(filepath.Join(mainRepo, sharedVaultConfigRelPath), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	// Sibling worktree OUTSIDE the main repo: upward walks never reach the
	// main checkout, and its branch predates the config commit (no config
	// in the worktree checkout).
	worktree := filepath.Join(base, "wt")
	worktreeGitDir := filepath.Join(mainGitDir, "worktrees", "wt")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(worktreeGitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitPtr := "gitdir: " + filepath.ToSlash(worktreeGitDir) + "\n"
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte(gitPtr), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeGitDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(worktree); err != nil {
		t.Fatal(err)
	}

	oldVault := vaultDir
	vaultDir = ""
	defer func() { vaultDir = oldVault }()

	got := resolveVaultDir()
	if resolved, rerr := filepath.EvalSymlinks(got); rerr == nil {
		got = resolved
	}
	want := filepath.Join(mainGitDir, "paivot", "nd-vault")
	if resolved, rerr := filepath.EvalSymlinks(want); rerr == nil {
		want = resolved
	}
	if got != want {
		t.Fatalf("resolveVaultDir() from sibling worktree = %q, want shared vault %q", got, want)
	}
}

func TestResolveVaultDir_FallsBackToInitializedSharedVault(t *testing.T) {
	base := t.TempDir()

	// Main repo, NO shared config anywhere, but the shared vault under the
	// git common dir is initialized -- it is the live source of record.
	mainRepo := filepath.Join(base, "main-repo")
	mainGitDir := filepath.Join(mainRepo, ".git")
	sharedVault := filepath.Join(mainGitDir, "paivot", "nd-vault")
	if err := os.MkdirAll(sharedVault, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedVault, ".nd.yaml"), []byte("vault: ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A stale legacy local vault must lose to the initialized shared vault.
	legacyVault := filepath.Join(mainRepo, ".vault")
	if err := os.MkdirAll(legacyVault, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyVault, ".nd.yaml"), []byte("vault: legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(mainRepo); err != nil {
		t.Fatal(err)
	}

	oldVault := vaultDir
	vaultDir = ""
	defer func() { vaultDir = oldVault }()

	got := resolveVaultDir()
	if resolved, rerr := filepath.EvalSymlinks(got); rerr == nil {
		got = resolved
	}
	want := sharedVault
	if resolved, rerr := filepath.EvalSymlinks(want); rerr == nil {
		want = resolved
	}
	if got != want {
		t.Fatalf("resolveVaultDir() = %q, want initialized shared vault %q", got, want)
	}
}

func TestVaultDivergence(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, ".vault")
	if err := os.MkdirAll(vault, 0o755); err != nil {
		t.Fatal(err)
	}

	if vaultDivergence(vault, vault) {
		t.Error("identical paths should not diverge")
	}
	if vaultDivergence(filepath.Join(dir, "sub", "..", ".vault"), vault) {
		t.Error("lexically equivalent paths should not diverge")
	}

	other := filepath.Join(dir, "shared", "nd-vault")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if !vaultDivergence(other, vault) {
		t.Error("different paths should diverge")
	}

	if vaultDivergence("", vault) || vaultDivergence(other, "") {
		t.Error("empty paths should never diverge")
	}

	// Symlink to the same vault must not be reported as divergence.
	link := filepath.Join(dir, "link-vault")
	if err := os.Symlink(vault, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	if vaultDivergence(link, vault) {
		t.Error("symlinked path to the same vault should not diverge")
	}
}

// captureDivergenceWarning runs resolveVaultDir with a fresh warning state and
// returns whatever was written to stderr.
func captureDivergenceWarning(t *testing.T) string {
	t.Helper()

	oldWarned := vaultDivergenceWarned
	vaultDivergenceWarned = false
	defer func() { vaultDivergenceWarned = oldWarned }()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	resolveVaultDir()

	_ = w.Close()
	os.Stderr = oldStderr
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// setupStaleLocalVault creates a repo whose initialized shared vault wins
// resolution while a stale local .vault still exists in the checkout, then
// chdirs into it. Returns a cleanup-managed environment via t.Cleanup.
func setupStaleLocalVault(t *testing.T) {
	t.Helper()

	base := t.TempDir()
	mainRepo := filepath.Join(base, "main-repo")
	sharedVault := filepath.Join(mainRepo, ".git", "paivot", "nd-vault")
	if err := os.MkdirAll(sharedVault, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedVault, ".nd.yaml"), []byte("vault: ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleVault := filepath.Join(mainRepo, ".vault")
	if err := os.MkdirAll(staleVault, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleVault, ".nd.yaml"), []byte("vault: stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := os.Chdir(mainRepo); err != nil {
		t.Fatal(err)
	}

	oldVault := vaultDir
	vaultDir = ""
	t.Cleanup(func() { vaultDir = oldVault })
}

func TestResolveVaultDir_WarnsOnStaleLocalVault(t *testing.T) {
	setupStaleLocalVault(t)

	out := captureDivergenceWarning(t)
	if !strings.Contains(out, "ignoring local .vault at") {
		t.Errorf("expected stale-vault warning on stderr, got %q", out)
	}
	if !strings.Contains(out, "stale worktree copy?") {
		t.Errorf("warning should mention stale worktree copy, got %q", out)
	}
}

func TestResolveVaultDir_QuietSuppressesDivergenceWarning(t *testing.T) {
	setupStaleLocalVault(t)

	oldQuiet := quiet
	quiet = true
	defer func() { quiet = oldQuiet }()

	out := captureDivergenceWarning(t)
	if out != "" {
		t.Errorf("quiet should suppress the divergence warning, got %q", out)
	}
}

func TestResolveVaultDir_NoWarningWhenLocalVaultResolved(t *testing.T) {
	base := t.TempDir()
	projectRoot := filepath.Join(base, "repo")
	if err := os.MkdirAll(filepath.Join(projectRoot, ".vault"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".vault", ".nd.yaml"), []byte("version: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}

	oldVault := vaultDir
	vaultDir = ""
	defer func() { vaultDir = oldVault }()

	out := captureDivergenceWarning(t)
	if out != "" {
		t.Errorf("no warning expected when the local .vault is the resolved vault, got %q", out)
	}
}

func setupSharedWorktree(t *testing.T) (projectRoot, sharedVault string) {
	t.Helper()

	base := t.TempDir()
	projectRoot = filepath.Join(base, "repo")
	gitDir := filepath.Join(base, "gitdir", "worktrees", "story")
	commonDir := filepath.Join(base, "gitdir")
	sharedVault = filepath.Join(commonDir, "shared", "nd-vault")

	if err := os.MkdirAll(filepath.Join(projectRoot, ".vault"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "# nd shared-worktree state\nmode: git_common_dir\npath: shared/nd-vault\n"
	if err := os.WriteFile(filepath.Join(projectRoot, ".vault", ".nd-shared.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sharedVault, 0o755); err != nil {
		t.Fatal(err)
	}

	gitPtr := "gitdir: " + filepath.ToSlash(gitDir) + "\n"
	if err := os.WriteFile(filepath.Join(projectRoot, ".git"), []byte(gitPtr), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return projectRoot, sharedVault
}
