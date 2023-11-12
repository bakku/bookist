require "rails_helper"

RSpec.describe PreparePasswordResetJob do
  subject { described_class.new }

  let(:user) { users(:joe) }

  it "does nothing if user does not exist" do
    expect do
      subject.perform("unknown@example.com", "en")
    end.not_to change(ApplicationMailer.deliveries, :count).from(0)
  end

  it "sends out an email if the user exists" do
    expect do
      subject.perform(user.email, "en")
    end.to change(ApplicationMailer.deliveries, :count).from(0).to(1)

    user.reload

    expect(user.password_reset_token).not_to be(nil)
    expect(user.password_reset_token_created_at).to be_within(1.minute).of(Time.current)
  end
end
