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
end
