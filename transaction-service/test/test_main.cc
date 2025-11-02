#define DROGON_TEST_MAIN
#include <drogon/drogon_test.h>
#include <drogon/drogon.h>
#include <json/json.h>

using namespace drogon;

static void configureApp() {
  app().loadConfigFile("./config.json");
}

DROGON_TEST(TransactionsCrud)
{
    configureApp();

    auto client = HttpClient::newHttpClient("http://localhost:5555");

    // Generate UUIDs for test
    auto uid = utils::getUuid();
    auto mid = utils::getUuid();
    auto oid = utils::getUuid();

    // Create
    Json::Value createBody;
    createBody["user_id"] = uid;
    createBody["market_id"] = mid;
    createBody["option_id"] = oid;
    createBody["transaction_type"] = "BUY";
    createBody["number_of_shares"] = "10";
    createBody["price_per_share"] = "1.23";

    auto reqCreate = HttpRequest::newHttpJsonRequest(createBody);
    reqCreate->setPath("/transactions");
    reqCreate->setMethod(Post);

    client->sendRequest(reqCreate, [TEST_CTX](ReqResult result, const HttpResponsePtr &resp) {
        REQUIRE(result == ReqResult::Ok);
        REQUIRE(resp != nullptr);
        CHECK(resp->getStatusCode() == k201Created);
        auto json = resp->getJsonObject();
        REQUIRE(json != nullptr);
        CHECK(json->isMember("id"));
        CHECK(json->isMember("created_at"));
        CHECK((*json)["transaction_type"].asString() == "BUY");
    });
}

DROGON_TEST(TransactionsGetList)
{
    configureApp();
    auto client = HttpClient::newHttpClient("http://localhost:5555");
    auto req = HttpRequest::newHttpRequest();
    req->setPath("/transactions");
    req->setMethod(Get);
    client->sendRequest(req, [TEST_CTX](ReqResult result, const HttpResponsePtr &resp) {
        REQUIRE(result == ReqResult::Ok);
        REQUIRE(resp != nullptr);
        CHECK(resp->getStatusCode() == k200OK);
        auto json = resp->getJsonObject();
        REQUIRE(json != nullptr);
        CHECK(json->isArray());
    });
}

int main(int argc, char** argv) 
{
    std::promise<void> p1;
    std::future<void> f1 = p1.get_future();

    std::thread thr([&]() {
        app().getLoop()->queueInLoop([&p1]() { p1.set_value(); });
        app().run();
    });

    f1.get();
    int status = drogon::test::run(argc, argv);

    app().getLoop()->queueInLoop([]() { app().quit(); });
    thr.join();
    return status;
}
