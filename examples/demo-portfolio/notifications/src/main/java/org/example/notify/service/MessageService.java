package org.example.notify.service;
import org.springframework.stereotype.Service;
import org.springframework.jms.annotation.JmsListener;
import org.example.notify.repository.CaseReadRepository;
@Service
public class MessageService {
    private CaseReadRepository repo;
    public void send(String b) { }
    @JmsListener(destination = "jms/queue/CaseResolved")
    public void onCaseResolved(String msg) { repo.load(msg); }
}
