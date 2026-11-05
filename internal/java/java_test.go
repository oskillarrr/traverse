package java

import (
	"testing"

	"github.com/oskillarrr/traverse/internal/lang"
)

const src = `package p;
import java.util.*;
/** This class does things */
@Path("/r")
public class R extends Base implements A, b.c.D {
    @Inject private Svc svc;
    private Foo foo = new Foo();
    @GET
    @Path(
       "/{id}")
    public String get(@PathParam("id") String id) { return "x"; }
    public void helper() { if (x) { y(); } }
    @GetMapping(value = {"/{id}", "/by-id/{id}"}, produces = "json")
    public String two() { return null; }
    static class Inner { void m() {} }
    enum E { A, B }
}
record Point(int x, int y) implements Shape {}
`

func TestParse(t *testing.T) {
	f := Parse(lang.Strip(src, lang.CStyle))
	if f.Package != "p" || len(f.Imports) != 1 || !f.Imports[0].Wildcard {
		t.Fatalf("package/imports: %+v", f)
	}
	byName := map[string]*Type{}
	for _, ty := range f.Types {
		byName[ty.FQN] = ty
	}
	r := byName["p.R"]
	if r == nil || len(r.Annotations) != 1 || r.Annotations[0].First("value", "") != "/r" {
		t.Fatalf("class annotations: %+v", r)
	}
	if got := r.Extends; len(got) != 3 || got[2] != "b.c.D" {
		t.Errorf("extends: %v", got)
	}
	if byName["p.R.Inner"] == nil || byName["p.R.E"] == nil || byName["p.Point"] == nil {
		t.Errorf("nested/enum/record types missing: %v", byName)
	}
	methods := map[string]Method{}
	for _, m := range r.Methods {
		methods[m.Name] = m
	}
	if a, ok := Has(methods["get"].Annotations, "Path"); !ok || a.First("value", "") != "/{id}" {
		t.Errorf("multi-line @Path not attached to get(): %+v", methods["get"])
	}
	if len(methods["helper"].Annotations) != 0 {
		t.Errorf("helper() must not inherit annotations")
	}
	if a, _ := Has(methods["two"].Annotations, "GetMapping"); len(a.Values("value", "path", "")) != 2 {
		t.Errorf("array values: %v", a.Values("value"))
	}
	var injected bool
	for _, fd := range r.Fields {
		if fd.Type == "Svc" && len(fd.Annotations) == 1 {
			injected = true
		}
	}
	if !injected {
		t.Errorf("injected field not found: %+v", r.Fields)
	}
}

const csSrc = `using System;
using Microsoft.AspNetCore.Mvc;
using Shop.Core.Services;

namespace Shop.Web.Controllers
{
    [ApiController]
    [Route("api/[controller]")]
    public class OrdersController : ControllerBase, IDisposable where T : class
    {
        private readonly IOrderService _orders;
        public OrdersController(IOrderService orders) { _orders = orders; }

        [HttpGet("{id}")]
        public async Task<IActionResult> Get(int id) => Ok(await _orders.Find(id));

        [HttpPost, Route("~/v2/orders")]
        public IActionResult Create([FromBody] OrderDto dto) { return Ok(); }

        public int Count { get; set; }
    }
    public record OrderDto(int Id, string Name);
}
`

func TestParseCSharp(t *testing.T) {
	f := ParseDialect(lang.Strip(csSrc, lang.CStyle), CSharp)
	if f.Package != "Shop.Web.Controllers" || len(f.Imports) != 3 || !f.Imports[2].Wildcard {
		t.Fatalf("namespace/usings: %q %+v", f.Package, f.Imports)
	}
	var c *Type
	for _, ty := range f.Types {
		if ty.Name == "OrdersController" {
			c = ty
		}
	}
	if c == nil || len(c.Annotations) != 2 || c.Annotations[1].First("") != "api/[controller]" {
		t.Fatalf("class attributes: %+v", c)
	}
	if len(c.Extends) != 2 || c.Extends[0] != "ControllerBase" {
		t.Errorf("base types: %v", c.Extends)
	}
	m := map[string]Method{}
	for _, x := range c.Methods {
		m[x.Name] = x
	}
	if a, ok := Has(m["Get"].Annotations, "HttpGet"); !ok || a.First("") != "{id}" || !m["Get"].Public {
		t.Errorf("Get: %+v", m["Get"])
	}
	if _, ok := Has(m["Create"].Annotations, "Route"); !ok {
		t.Errorf("Create attributes: %+v", m["Create"])
	}
	if len(f.Types) != 2 {
		t.Errorf("types: %d", len(f.Types))
	}
}

const ktSrc = `package org.example.vet

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController
import org.example.model.*

@RestController
class VetController(private val vetRepository: VetRepository) : BaseController() {
    val cache = mutableMapOf<String, String>()

    @GetMapping("/vets")
    fun showVetList(model: Model): String {
        return "vets/vetList"
    }
}

data class Vets(val vetList: List<Vet>)

enum class Kind { A, B }

object Registry {
    fun all() = listOf<String>()
}
`

func TestParseKotlin(t *testing.T) {
	f := ParseDialect(lang.Strip(ktSrc, lang.CStyle), Kotlin)
	if f.Package != "org.example.vet" || len(f.Imports) != 3 || !f.Imports[2].Wildcard {
		t.Fatalf("package/imports: %q %+v", f.Package, f.Imports)
	}
	names := map[string]*Type{}
	for _, ty := range f.Types {
		names[ty.Name] = ty
	}
	for _, n := range []string{"VetController", "Vets", "Kind", "Registry"} {
		if names[n] == nil || !names[n].TopLevel {
			t.Errorf("missing top-level %s: %v", n, names)
		}
	}
	vc := names["VetController"]
	if len(vc.Extends) != 1 || vc.Extends[0] != "BaseController" {
		t.Errorf("extends: %v", vc.Extends)
	}
	if len(vc.Methods) != 1 || vc.Methods[0].Name != "showVetList" {
		t.Fatalf("methods: %+v", vc.Methods)
	}
	if a, ok := Has(vc.Methods[0].Annotations, "GetMapping"); !ok || a.First("value", "") != "/vets" {
		t.Errorf("annotation: %+v", vc.Methods[0])
	}
}
