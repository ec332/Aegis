#include "TransactionRepository.h"
#include <drogon/drogon.h>

using drogon_model::transaction::Transactions;

// Find all
drogon::Task<std::vector<Transactions>> TransactionRepository::findAll() {
  co_return co_await mapper_.findAll();
}

// Find by id
drogon::Task<Transactions> TransactionRepository::findById(const std::string& id) {
  co_return co_await mapper_.findByPrimaryKey(id);
}

// Insert
drogon::Task<Transactions> TransactionRepository::insert(const Transactions& t) {
  co_return co_await mapper_.insert(t);
}

// Update
drogon::Task<size_t> TransactionRepository::update(const Transactions& t) {
  co_return co_await mapper_.update(t);
}

// Delete
drogon::Task<size_t> TransactionRepository::deleteById(const std::string& id) {
  co_return co_await mapper_.deleteByPrimaryKey(id);
}