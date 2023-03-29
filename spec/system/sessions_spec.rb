require "rails_helper"

RSpec.describe "Session Flows" do
  let(:user) { users(:joe) }

  describe "Login Flow" do
    it "logs in existing users and redirects them to the me page" do
      visit root_path
      expect(page).to have_selector("h2", text: "Sign in")

      fill_in "Username", with: user.username
      fill_in "Password", with: "test1234"
      click_on "Sign in"

      expect(page).to have_selector("h1", text: "Hi #{user.username}")
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
end
