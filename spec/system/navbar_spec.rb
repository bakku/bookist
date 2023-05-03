require "rails_helper"

RSpec.describe "Navbar" do
  let(:user) { users(:joe) }

  it "does not show the navbar on pages like the login page" do
    visit login_path

    expect(page).not_to have_selector("[data-testid='navbar']")
  end

  it "shows the navbar on pages like the dashboard page" do
    visit root_path
    expect(page).to have_selector("h2", text: "Sign in")

    fill_in "Username", with: user.username
    fill_in "Password", with: "test1234"
    click_on "Sign in"

    expect(page).to have_selector("[data-testid='navbar']")
  end
end
