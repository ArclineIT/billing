class ApplicationMailer < ActionMailer::Base
  default from: ENV.fetch("SMTP_FROM", "support@arcline.it")
  layout "mailer"
end
