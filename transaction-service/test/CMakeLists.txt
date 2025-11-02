cmake_minimum_required(VERSION 3.5)
project(transaction-service_test CXX)

add_executable(${PROJECT_NAME} test_main.cc)
# Ensure test binary is placed under test directory
set_target_properties(${PROJECT_NAME} PROPERTIES RUNTIME_OUTPUT_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR})

# ##############################################################################
# If you include the drogon source code locally in your project, use this method
# to add drogon 
# target_link_libraries(${PROJECT_NAME} PRIVATE drogon)
#
# and comment out the following lines
target_link_libraries(${PROJECT_NAME} PRIVATE Drogon::Drogon)

ParseAndAddDrogonTests(${PROJECT_NAME})
