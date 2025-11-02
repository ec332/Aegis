#include "controllers/TransactionController.h"
#include <string>
#include <drogon/drogon.h>
#include <drogon/orm/Exception.h>
#include "models/Transactions.h"

using drogon::orm::UnexpectedRows;
using drogon::orm::DrogonDbException;

void TransactionController::getOne(const HttpRequestPtr &req,
                         std::function<void(const HttpResponsePtr &)> &&callback,
                         std::string &&id)
{
    auto service = makeService();
    drogon::async_run([service, id = std::move(id), cb = std::move(callback)]() mutable -> drogon::Task<void> {
        try {
            auto t = co_await service.findById(id);
            auto resp = HttpResponse::newHttpJsonResponse(t.toJson());
            resp->setStatusCode(k200OK);
            cb(resp);
        } catch (const DrogonDbException &e) {
            auto resp = HttpResponse::newHttpJsonResponse(Json::Value(Json::objectValue));
            if (dynamic_cast<const UnexpectedRows *>(&e) != nullptr) {
                resp->setStatusCode(k404NotFound);
            } else {
                Json::Value err;
                err["error"] = e.base().what();
                resp = HttpResponse::newHttpJsonResponse(err);
                resp->setStatusCode(k500InternalServerError);
            }
            cb(resp);
        }
        co_return;
    });
}

void TransactionController::get(const HttpRequestPtr &req,
                      std::function<void(const HttpResponsePtr &)> &&callback)
{
    auto service = makeService();
    drogon::async_run([service, cb = std::move(callback)]() mutable -> drogon::Task<void> {
        try {
            auto list = co_await service.findAll();
            Json::Value arr(Json::arrayValue);
            for (const auto &t : list) {
                arr.append(t.toJson());
            }
            auto resp = HttpResponse::newHttpJsonResponse(arr);
            resp->setStatusCode(k200OK);
            cb(resp);
        } catch (const DrogonDbException &e) {
            Json::Value err;
            err["error"] = e.base().what();
            auto resp = HttpResponse::newHttpJsonResponse(err);
            resp->setStatusCode(k500InternalServerError);
            cb(resp);
        }
        co_return;
    });
}

void TransactionController::create(const HttpRequestPtr &req,
                         std::function<void(const HttpResponsePtr &)> &&callback)
{
    auto json = req->getJsonObject();
    if (!json) {
        Json::Value err;
        err["error"] = "Invalid JSON body";
        auto resp = HttpResponse::newHttpJsonResponse(err);
        resp->setStatusCode(k400BadRequest);
        callback(resp);
        return;
    }

    auto service = makeService();
    drogon::async_run([service, body = *json, cb = std::move(callback)]() mutable -> drogon::Task<void> {
        try {
            auto inserted = co_await service.create(body);
            auto resp = HttpResponse::newHttpJsonResponse(inserted.toJson());
            resp->setStatusCode(k201Created);
            cb(resp);
        } catch (const std::runtime_error &e) {
            Json::Value err;
            err["error"] = e.what();
            auto resp = HttpResponse::newHttpJsonResponse(err);
            resp->setStatusCode(k400BadRequest);
            cb(resp);
        } catch (const DrogonDbException &e) {
            Json::Value err;
            err["error"] = e.base().what();
            auto resp = HttpResponse::newHttpJsonResponse(err);
            resp->setStatusCode(k500InternalServerError);
            cb(resp);
        }
        co_return;
    });
}

void TransactionController::updateOne(const HttpRequestPtr &req,
                            std::function<void(const HttpResponsePtr &)> &&callback,
                            std::string &&id)
{
    auto json = req->getJsonObject();
    if (!json) {
        Json::Value err;
        err["error"] = "Invalid JSON body";
        auto resp = HttpResponse::newHttpJsonResponse(err);
        resp->setStatusCode(k400BadRequest);
        callback(resp);
        return;
    }

    auto service = makeService();
    drogon::async_run([service, id = std::move(id), body = *json, cb = std::move(callback)]() mutable -> drogon::Task<void> {
        try {
            auto updated = co_await service.update(id, body);
            auto resp = HttpResponse::newHttpJsonResponse(updated.toJson());
            resp->setStatusCode(k200OK);
            cb(resp);
        } catch (const std::runtime_error &e) {
            Json::Value err;
            err["error"] = e.what();
            auto resp = HttpResponse::newHttpJsonResponse(err);
            resp->setStatusCode(k400BadRequest);
            cb(resp);
        } catch (const DrogonDbException &e) {
            Json::Value err;
            err["error"] = e.base().what();
            auto resp = HttpResponse::newHttpJsonResponse(err);
            resp->setStatusCode(k500InternalServerError);
            cb(resp);
        }
        co_return;
    });
}

void TransactionController::deleteOne(const HttpRequestPtr &req,
                            std::function<void(const HttpResponsePtr &)> &&callback,
                            std::string &&id)
{
    auto service = makeService();
    drogon::async_run([service, id = std::move(id), cb = std::move(callback)]() mutable -> drogon::Task<void> {
        try {
            auto affected = co_await service.deleteById(id);
            if (affected == 0) {
                auto resp = HttpResponse::newHttpJsonResponse(Json::Value(Json::objectValue));
                resp->setStatusCode(k404NotFound);
                cb(resp);
            } else {
                auto resp = HttpResponse::newHttpJsonResponse(Json::Value(Json::objectValue));
                resp->setStatusCode(k204NoContent);
                cb(resp);
            }
        } catch (const DrogonDbException &e) {
            auto resp = HttpResponse::newHttpJsonResponse(Json::Value(Json::objectValue));
            if (dynamic_cast<const UnexpectedRows *>(&e) != nullptr) {
                resp->setStatusCode(k404NotFound);
            } else {
                Json::Value err;
                err["error"] = e.base().what();
                resp = HttpResponse::newHttpJsonResponse(err);
                resp->setStatusCode(k500InternalServerError);
            }
            cb(resp);
        }
        co_return;
    });
}
