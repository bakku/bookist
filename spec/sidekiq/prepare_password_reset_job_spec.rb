# frozen_string_literal: true

require "rails_helper"

RSpec.describe PreparePasswordResetJob do
  let(:user) { users(:joe) }

  it "does nothing if user does not exist" do
    expect do
      described_class.new.perform("unknown@example.com", "en")
    end.to_not change(ApplicationMailer.deliveries, :count).from(0)
  end

  it "sends out an email if the user exists" do
    expect do
      described_class.new.perform(user.email, "en")
    end.to change(ApplicationMailer.deliveries, :count).from(0).to(1)

    user.reload

    expect(user.password_reset_token).to_not be_nil
    expect(user.password_reset_token_created_at).to be_within(1.minute).of(Time.current)
  end
end
