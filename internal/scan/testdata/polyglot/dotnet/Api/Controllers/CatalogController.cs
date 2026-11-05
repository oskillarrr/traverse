using Microsoft.AspNetCore.Mvc;
namespace Shop.Api.Controllers
{
    [ApiController]
    [Route("api/[controller]")]
    public class CatalogController : ControllerBase
    {
        [HttpGet("{id}")]
        public IActionResult Get(int id) => Ok();
    }
}
