require "rails_helper"

RSpec.describe User do
  let(:user) { users(:joe) }

  describe "has_secure_password" do
    it "verifies the password" do
      user.password = "12345"
      user.password_confirmation = "123456"

      expect(user.valid?).to be_falsey
    end

    it "enables authentication of users" do
      expect(user.authenticate("test1234")).to be_truthy
      expect(user.authenticate("test12345")).to be_falsey
    end
  end

  describe "validations" do
    it "is valid with all fields present" do
      expect(user.valid?).to be_truthy
    end

    it "requires the username to be present" do
      user.username = nil
      expect(user.valid?).to be_falsey
    end

    it "requires the email to be present" do
      user.username = nil
      expect(user.valid?).to be_falsey
    end

    it "requires the username to be unique" do
      user.update!(username: "duplicated")

      paul = users(:paul).tap do |u|
        u.username = "duplicated"
      end

      expect(paul.valid?).to be_falsey
    end

    it "requires the email to be unique" do
      user.update!(email: "duplicated")

      paul = users(:paul).tap do |u|
        u.email = "DupliCated"
      end

      expect(paul.valid?).to be_falsey
    end
  end
end
