package org.example.notify.api;
import org.springframework.web.bind.annotation.*;
import org.example.notify.service.MessageService;
@RestController
@RequestMapping("/api/messages")
public class MessageController {
    private final MessageService service;
    public MessageController(MessageService s) { this.service = s; }
    @PostMapping("/send")
    public void send(@RequestBody String body) { service.send(body); }
    @GetMapping(value = {"/{id}", "/by-id/{id}"})
    public String get(@PathVariable String id) { return id; }
}
