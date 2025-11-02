#pragma once
#include <drogon/orm/DbClient.h>
#include <drogon/orm/CoroMapper.h>
#include "models/Transactions.h"
#include <vector>

class TransactionRepository {
public:
  explicit TransactionRepository(const drogon::orm::DbClientPtr& client)
    : client_(client), mapper_(client) {}

  drogon::Task<std::vector<drogon_model::transaction::Transactions>> findAll();
  drogon::Task<drogon_model::transaction::Transactions> findById(const std::string& id);
  drogon::Task<drogon_model::transaction::Transactions> insert(const drogon_model::transaction::Transactions& t);
  drogon::Task<size_t> update(const drogon_model::transaction::Transactions& t);
  drogon::Task<size_t> deleteById(const std::string& id);

private:
  drogon::orm::DbClientPtr client_;
  drogon::orm::CoroMapper<drogon_model::transaction::Transactions> mapper_;
};