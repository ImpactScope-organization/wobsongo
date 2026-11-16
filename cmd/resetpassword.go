package cmd

import (
	"errors"
	"os"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var (
	resetPasswordEmail    string
	resetPasswordPassword string
	resetPasswordApply    bool
)

var resetPasswordCmd = &cobra.Command{
	Use:   "reset-password",
	Short: "Reset (override) the password of an existing account",
	Long: "Sets a new password for the account with the given email, without\n" +
		"needing the current one. Intended as an operator-side recovery until a\n" +
		"\"forgot password\" flow exists.\n\n" +
		"Refresh tokens issued before the reset stop working, so the user is\n" +
		"signed out of every session once their access token expires.\n\n" +
		"Without --apply, this is a dry run: the password is validated, but\n" +
		"nothing is changed.",
	Run: runResetPassword,
}

func init() {
	resetPasswordCmd.Flags().
		StringVar(&resetPasswordEmail, "email", "", "Email of the account to reset (required)")
	resetPasswordCmd.Flags().
		StringVar(&resetPasswordPassword, "password", "", "New password (required)")
	resetPasswordCmd.Flags().
		BoolVar(&resetPasswordApply, "apply", false, "Actually reset the password (default is a dry run)")

	_ = resetPasswordCmd.MarkFlagRequired("email")
	_ = resetPasswordCmd.MarkFlagRequired("password")
}

func runResetPassword(cmd *cobra.Command, _ []string) {
	if !auth.PasswordValid(resetPasswordPassword) {
		cmd.PrintErrf(
			"Password must be at least %d characters and at most %d bytes.\n",
			auth.PasswordMinLength, auth.PasswordMaxBytes,
		)
		os.Exit(1)
		return
	}

	if !resetPasswordApply {
		cmd.Printf("Dry run: would reset the password of %q\n", resetPasswordEmail)
		cmd.Println("Re-run with --apply to actually reset the password.")
		return
	}

	ctx := cmd.Context()
	config := internal.NewConfig(EnvFile)

	pool, err := pgxpool.New(ctx, config.PostgresURI)
	if err != nil {
		cmd.PrintErrf("Failed to connect to database: %s\n", err.Error())
		os.Exit(1)
		return
	}
	defer pool.Close()

	userRepo := repo.NewUserRepo(db.New(pool), pool)
	authSvc := service.NewAuthService(
		userRepo,
		auth.New(config.JWTSecret, config.JWTAccessTTL, config.JWTRefreshTTL),
	)

	user, err := authSvc.ResetPassword(ctx, resetPasswordEmail, resetPasswordPassword)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			cmd.PrintErrf("No account with email %q exists.\n", resetPasswordEmail)
		} else {
			cmd.PrintErrf("Failed to reset password: %s\n", err.Error())
		}
		os.Exit(1)
		return
	}

	cmd.Printf("Reset password for %s (email=%q, name=%q)\n", user.ID, user.Email, user.Name)
}
