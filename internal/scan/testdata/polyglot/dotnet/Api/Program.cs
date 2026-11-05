var app = WebApplication.Create(args);
app.MapGet("/health/ready", () => "ok");
