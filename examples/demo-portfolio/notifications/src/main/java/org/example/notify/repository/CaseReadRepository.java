package org.example.notify.repository;
import org.springframework.stereotype.Repository;
import org.springframework.jdbc.core.JdbcTemplate;
@Repository
public class CaseReadRepository {
    private JdbcTemplate jdbc;
    public void load(String id) { jdbc.queryForList("select STATUS from CASES.EXPEDIENTE where ID = ?", id); }
}
