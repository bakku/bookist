class PreparePasswordResetJob
  include Sidekiq::Job

  def perform(email, locale)
    user = User.find_by(email: email)
    authority = PasswordResetAuthority.new

    if user.present?
      authority.prepare_reset!(user)

      UserMailer.reset_password_instructions(user, locale).deliver_now
    end
  end
end
