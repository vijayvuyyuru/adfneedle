import asyncio
from typing import Any, ClassVar, Final, Mapping, Optional, Sequence

from typing_extensions import Self
from viam.components.sensor import *
from viam.module.module import Module
from viam.proto.app.robot import ComponentConfig
from viam.proto.common import ResourceName
from viam.resource.base import ResourceBase
from viam.resource.easy_resource import EasyResource
from viam.resource.types import Model, ModelFamily
from viam.utils import SensorReading, struct_to_dict
from viam.logging import getLogger
from pymongo import MongoClient
import json




LOGGER = getLogger(__name__)

def get_value_from_json(json_file, key):
   try:
       with open(json_file) as f:
           data = json.load(f)
           return data[key]
   except Exception as e:
       print("Error: ", e)


class Adfneedle(Sensor, EasyResource):
    MODEL: ClassVar[Model] = Model(
        ModelFamily("viam-data-ml", "sensor"), "adfneedle"
    )
    
    limit = 0
    secret_path = ""

    @classmethod
    def new(
        cls, config: ComponentConfig, dependencies: Mapping[ResourceName, ResourceBase]
    ) -> Self:
        """This method creates a new instance of this Sensor component.
        The default implementation sets the name from the `config` parameter and then calls `reconfigure`.

        Args:
            config (ComponentConfig): The configuration for this resource
            dependencies (Mapping[ResourceName, ResourceBase]): The dependencies (both implicit and explicit)

        Returns:
            Self: The resource
        """
        return super().new(config, dependencies)

    def reconfigure(
        self, config: ComponentConfig, dependencies: Mapping[ResourceName, ResourceBase]
    ):
        # self.validate_config("", config)
        attrs = struct_to_dict(config.attributes)
        if attrs.get("limit") is not None:
            self.limit = int(attrs.get("limit"))
        else:
            raise Exception("limit must be specified")
        if attrs.get("secret_path") is not None:
            self.secret_path = str(attrs.get("secret_path"))
        else:
            raise Exception("secret path must be specified")
        LOGGER.debug(f"Using limit: {self.limit}")
        """This method allows you to dynamically update your service when it receives a new `config` object.

        Args:
            config (ComponentConfig): The new configuration
            dependencies (Mapping[ResourceName, ResourceBase]): Any dependencies (both implicit and explicit)
        """
        return super().reconfigure(config, dependencies)

    async def get_readings(
        self,
        *,
        extra: Optional[Mapping[str, Any]] = None,
        timeout: Optional[float] = None,
        **kwargs
    ) -> Mapping[str, SensorReading]:
        client = MongoClient(get_value_from_json(self.secret_path, "url"))
        result = client['syncDB']['data_federations'].aggregate([
            {
                '$count': 'count'
            }
        ])
        count_dict = result.next()
        
        return {
            "limit": self.limit,
            "count": count_dict['count'],
            "usage": float(count_dict['count'])/float(self.limit)
        }



if __name__ == "__main__":
    asyncio.run(Module.run_from_registry())


