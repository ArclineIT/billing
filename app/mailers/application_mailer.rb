class ApplicationMailer < ActionMailer::Base
  default from: ENV.fetch("SMTP_FROM", "support@arclineit.com")
  layout "mailer"
end
