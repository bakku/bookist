require "rails_helper"

RSpec.describe "Session Flows" do
  let(:user) { users(:joe) }

  describe "Login Flow" do
    it "should login existing users and redirect them to the me page" do
      visit root_path
      assert_selector "h2", text: "Sign in"

      fill_in "Username", with: user.username
      fill_in "Password", with: "test1234"
      click_on "Sign in"

      assert_selector "h1", text: "Hi #{user.username}"
    end
  end
end
