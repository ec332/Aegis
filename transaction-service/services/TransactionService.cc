#include "TransactionService.h"
#include <drogon/drogon.h>
#include <trantor/utils/Date.h>

using drogon_model::transaction::Transactions;

// List all
drogon::Task<std::vector<Transactions>> TransactionService::findAll() {
  co_return co_await repo_.findAll();
}

// Get one
drogon::Task<Transactions> TransactionService::findById(const std::string& id) {
  co_return co_await repo_.findById(id);
}

// Create

drogon::Task<Transactions> TransactionService::create(const Json::Value& json) {
  std::string err;
  if (!Transactions::validateJsonForCreation(json, err)) {
    throw std::runtime_error(err);
  }
  Transactions t(json);
  // Ensure created_at is set if not provided
  if (!t.getCreatedAt()) {
    t.setCreatedAt(trantor::Date::date());
  }
  co_return co_await repo_.insert(t);
}

// Update

drogon::Task<Transactions> TransactionService::update(const std::string& id, const Json::Value& json) {
  std::string err;
  if (!Transactions::validateJsonForUpdate(json, err)) {
    throw std::runtime_error(err);
  }
  Transactions t(json);
  t.setId(id);
  // Preserve existing created_at or set default if missing
  if (!t.getCreatedAt()) {
    // fetch existing
    try {
      auto existing = co_await repo_.findById(id);
      if (existing.getCreatedAt()) {
        t.setCreatedAt(existing.getValueOfCreatedAt());
      } else {
        t.setCreatedAt(trantor::Date::date());
      }
    } catch (...) {
      t.setCreatedAt(trantor::Date::date());
    }
  }
  co_await repo_.update(t);
  co_return co_await repo_.findById(id);
}

// Delete
drogon::Task<size_t> TransactionService::deleteById(const std::string& id) {
  co_return co_await repo_.deleteById(id);
}