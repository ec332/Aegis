#ifndef TRANSACTION_CONTROLLER_H
#define TRANSACTION_CONTROLLER_H

#include <drogon/HttpController.h>
#include <string>
#include <drogon/drogon.h>
#include "services/TransactionService.h"
using namespace drogon;
class TransactionController : public drogon::HttpController<TransactionController>
{
  public:
    // Set lowercase base path for this controller
    static const std::string& path() {
        static const std::string p = "/transactions";
        return p;
    }

    METHOD_LIST_BEGIN
    // use METHOD_ADD to add your custom processing function here;
    ADD_METHOD_TO(TransactionController::getOne,"/transactions/{1}",Get,Options);
    ADD_METHOD_TO(TransactionController::get,"/transactions",Get,Options);
    ADD_METHOD_TO(TransactionController::create,"/transactions",Post,Options);
    ADD_METHOD_TO(TransactionController::updateOne,"/transactions/{1}",Put,Options);
    //ADD_METHOD_TO(TransactionController::update,"/transactions",Put,Options);
    ADD_METHOD_TO(TransactionController::deleteOne,"/transactions/{1}",Delete,Options);
    METHOD_LIST_END

    void getOne(const HttpRequestPtr &req,
                std::function<void(const HttpResponsePtr &)> &&callback,
                std::string &&id);
    void updateOne(const HttpRequestPtr &req,
                std::function<void(const HttpResponsePtr &)> &&callback,
                std::string &&id);
    void deleteOne(const HttpRequestPtr &req,
                   std::function<void(const HttpResponsePtr &)> &&callback,
                   std::string &&id);
    void get(const HttpRequestPtr &req,
             std::function<void(const HttpResponsePtr &)> &&callback);
    void create(const HttpRequestPtr &req,
                std::function<void(const HttpResponsePtr &)> &&callback);

  private:
    TransactionService makeService() const {
      return TransactionService(drogon::app().getDbClient());
    }
};

#endif