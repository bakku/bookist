class UserMailer < ApplicationMailer
  def reset_password_instructions(user, locale)
    @user = user

    I18n.with_locale(locale) do
      mail(to: @user.email, subject: t(".subject"))
    end
  end
end
