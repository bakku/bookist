# frozen_string_literal: true

require "rails_helper"

RSpec.describe "Authentication Flows" do
  let(:user) { users(:joe) }

  describe "Login Flow" do
    it "logs in existing users and redirects them to the me page" do
      visit root_path
      expect(page).to have_selector("h2", text: "Sign in")

      fill_in "Username", with: user.username
      fill_in "Password", with: "test1234"
      click_on "Sign in"

      expect(page).to have_selector("h1", text: "Welcome back #{user.username}.")
    end

    it "displays an alert in case the user does not exist" do
      visit root_path
      expect(page).to have_selector("h2", text: "Sign in")

      fill_in "Username", with: "invalid"
      fill_in "Password", with: "test1234"
      click_on "Sign in"

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector("p", text: "Invalid username or password")
    end

    it "displays an alert in case the password is incorrect" do
      visit root_path
      expect(page).to have_selector("h2", text: "Sign in")

      fill_in "Username", with: user.username
      fill_in "Password", with: "test12345"
      click_on "Sign in"

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector("p", text: "Invalid username or password")
    end
  end

  describe "Logout Flow" do
    it "successfully logs out a logged in user" do
      visit root_path
      expect(page).to have_selector("h2", text: "Sign in")

      fill_in "Username", with: user.username
      fill_in "Password", with: "test1234"
      click_on "Sign in"

      find("[data-testid='navbar-menu-open']").click
      expect(page).to have_selector("[data-testid='navbar-menu-close']")

      click_on "Logout"
      expect(page).to have_selector("p", text: "Logout successful. Looking forward to see you again.")
    end
  end

  describe "Signup Flow" do
    it "creates a user and redirects the user to the login page to sign in" do
      visit root_path
      expect(page).to have_selector("h2", text: "Sign in")

      click_on "Sign up"
      expect(page).to have_selector("h2", text: "Sign up")

      fill_in "Username", with: "peter"
      fill_in "Email", with: "peter@example.org"
      fill_in "Password", with: "test1234"
      fill_in "Password Confirmation", with: "test1234"
      click_on "Sign up"

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector("p", text: "Your account was successfully created. You can now sign in.")

      fill_in "Username", with: "peter"
      fill_in "Password", with: "test1234"
      click_on "Sign in"

      expect(page).to have_selector("h1", text: "Welcome back peter.")
    end

    it "shows validation errors when creating a user" do
      visit new_user_path
      expect(page).to have_selector("h2", text: "Sign up")

      fill_in "Username", with: "peter"
      fill_in "Email", with: user.email.upcase
      fill_in "Password", with: "test1234"
      fill_in "Password Confirmation", with: "test1234"
      click_on "Sign up"

      expect(page).to have_selector("h2", text: "Sign up")
      expect(page).to have_selector("p", text: "Your account could not be created.")
      expect(page).to have_selector("span", text: "Email is already taken")
    end
  end

  describe "Password Reset Flow" do
    it "displays an error in case no token is given when resetting the password" do
      visit new_password_reset_path

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector(
        "p", text: "Your password reset link is invalid or has expired. Please restart the password reset process."
      )
    end

    it "displays an error in case the password reset token is not valid anymore when clicking on the email link" do
      visit root_path

      expect(page).to have_selector("a", text: "Reset it")
      click_on "Reset it"

      expect(page).to have_selector("h2", text: "Reset password")
      fill_in "Email", with: user.email
      click_on "Reset"

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector(
        "p",
        text: "We will send you instructions on how to reset your password in case your email exists in our database."
      )
      expect(PreparePasswordResetJob.jobs.size).to be(1)

      PreparePasswordResetJob.drain
      expect(ApplicationMailer.deliveries.count).to be(1)
      expect(user.reload.password_reset_token).to_not be_nil

      mail = ApplicationMailer.deliveries.first

      expect(mail.from).to eq(["bookist@bakku.dev"])
      expect(mail.to).to eq([user.email])
      expect(mail.subject).to eq("Reset your bookist password")
      expect(mail.body.decoded).to include("href=\"http://localhost.test/password_reset/new?token=#{user.password_reset_token}\"")

      # Act as if the reset has been requested a long time ago.
      user.update!(password_reset_token_created_at: 1.day.ago)

      visit new_password_reset_path(token: user.password_reset_token)

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector(
        "p", text: "Your password reset link is invalid or has expired. Please restart the password reset process."
      )
    end

    it "displays an error in case the password reset token is not valid anymore when submitting the reset form" do
      visit root_path

      expect(page).to have_selector("a", text: "Reset it")
      click_on "Reset it"

      expect(page).to have_selector("h2", text: "Reset password")
      fill_in "Email", with: user.email
      click_on "Reset"

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector(
        "p",
        text: "We will send you instructions on how to reset your password in case your email exists in our database."
      )
      expect(PreparePasswordResetJob.jobs.size).to be(1)

      PreparePasswordResetJob.drain
      expect(ApplicationMailer.deliveries.count).to be(1)
      expect(user.reload.password_reset_token).to_not be_nil

      mail = ApplicationMailer.deliveries.first

      expect(mail.from).to eq(["bookist@bakku.dev"])
      expect(mail.to).to eq([user.email])
      expect(mail.subject).to eq("Reset your bookist password")
      expect(mail.body.decoded).to include("href=\"http://localhost.test/password_reset/new?token=#{user.password_reset_token}\"")

      visit new_password_reset_path(token: user.password_reset_token)

      # Act as if the reset has been requested a long time ago.
      user.update!(password_reset_token_created_at: 1.day.ago)

      expect(page).to have_selector("h2", text: "Reset password")
      fill_in "New Password", with: "ResetPassword"
      fill_in "New Password Confirmation", with: "ResetPassword"
      click_on "Reset password"

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector(
        "p",
        text: "Your password could not be reset. Please restart the password reset process."
      )
    end

    it "successfully resets the password of a user" do
      visit root_path

      expect(page).to have_selector("a", text: "Reset it")
      click_on "Reset it"

      expect(page).to have_selector("h2", text: "Reset password")
      fill_in "Email", with: user.email
      click_on "Reset"

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector(
        "p",
        text: "We will send you instructions on how to reset your password in case your email exists in our database."
      )
      expect(PreparePasswordResetJob.jobs.size).to be(1)

      PreparePasswordResetJob.drain
      expect(ApplicationMailer.deliveries.count).to be(1)
      expect(user.reload.password_reset_token).to_not be_nil

      mail = ApplicationMailer.deliveries.first

      expect(mail.from).to eq(["bookist@bakku.dev"])
      expect(mail.to).to eq([user.email])
      expect(mail.subject).to eq("Reset your bookist password")
      expect(mail.body.decoded).to include("href=\"http://localhost.test/password_reset/new?token=#{user.password_reset_token}\"")

      visit new_password_reset_path(token: user.password_reset_token)

      expect(page).to have_selector("h2", text: "Reset password")
      fill_in "New Password", with: "ResetPassword"
      fill_in "New Password Confirmation", with: "ResetPassword"
      click_on "Reset password"

      expect(page).to have_selector("h2", text: "Sign in")
      expect(page).to have_selector("p", text: "Your password was successfully reset. Please sign in again.")
      fill_in "Username", with: user.username
      fill_in "Password", with: "ResetPassword"
      click_on "Sign in"

      expect(page).to have_selector("h1", text: "Welcome back #{user.username}.")
    end
  end
end
