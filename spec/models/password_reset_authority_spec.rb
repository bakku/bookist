# frozen_string_literal: true

require "rails_helper"

RSpec.describe PasswordResetAuthority do
  describe "#resettable?" do
    it "returns true if #password_reset_token_created_at is not older than the default token validity duration" do
      authority = described_class.new
      resettable_record = User.new(password_reset_token_created_at: 30.minutes.ago)
      expect(authority.resettable?(resettable_record)).to be_truthy
    end

    it "returns true if #password_reset_token_created_at is not older than the given token validity duration" do
      authority = described_class.new(2.hours)
      resettable_record = User.new(password_reset_token_created_at: 90.minutes.ago)
      expect(authority.resettable?(resettable_record)).to be_truthy
    end

    it "returns false if #password_reset_token_created_at is older than the default token validity duration" do
      authority = described_class.new
      resettable_record = User.new(password_reset_token_created_at: 65.minutes.ago)
      expect(authority.resettable?(resettable_record)).to be_falsey
    end

    it "returns false if #password_reset_token_created_at is older than the given token validity duration" do
      authority = described_class.new(30.minutes)
      resettable_record = User.new(password_reset_token_created_at: 35.minutes.ago)
      expect(authority.resettable?(resettable_record)).to be_falsey
    end
  end

  describe "#prepare_reset!" do
    it "assigns a new password_reset_token and sets the password_reset_token_created_at timestamp" do
      joe = users(:joe)

      authority = described_class.new
      authority.prepare_reset!(joe)

      joe.reload

      expect(joe.password_reset_token).to_not be_nil
      expect(joe.password_reset_token_created_at).to be_within(1.second).of(Time.current)
    end
  end

  describe "#complete_reset!" do
    it "returns false if token has expired" do
      joe = users(:joe)
      joe.update!(password_reset_token: SecureRandom.uuid, password_reset_token_created_at: 2.hours.ago)

      authority = described_class.new

      expect(authority.complete_reset!(joe, "resetted", "resetted"))
        .to be_falsey

      joe.reload

      expect(joe.authenticate("resetted")).to be_falsey
    end

    it "returns false if password and confirmation does not match" do
      joe = users(:joe)
      joe.update!(password_reset_token: SecureRandom.uuid, password_reset_token_created_at: 1.minute.ago)

      authority = described_class.new

      expect(authority.complete_reset!(joe, "resetted", "resetted2"))
        .to be_falsey

      joe.reload

      expect(joe.authenticate("resetted")).to be_falsey
    end

    it "returns true if password could be successfully reset" do
      joe = users(:joe)
      joe.update!(password_reset_token: SecureRandom.uuid, password_reset_token_created_at: 1.minute.ago)

      authority = described_class.new
      expect(authority.complete_reset!(joe, "resetted", "resetted"))
        .to be_truthy

      joe.reload

      expect(joe.authenticate("resetted")).to be_truthy
      expect(joe.password_reset_token).to be_nil
      expect(joe.password_reset_token_created_at).to be_nil
    end
  end
end
