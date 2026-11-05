using System.Web.UI;
using System.Data.SqlClient;
using System.ServiceModel;
public partial class _Default : Page {
    protected void Page_Load(object s, System.EventArgs e) {
        var q = "SELECT ID, STATUS FROM EXPEDIENTE WHERE STATUS = 'R'";
        var url = "https://cases.example/cases/api/expedientes/42";
        var json = new System.Net.Http.HttpClient().GetAsync(url);
    }
}
